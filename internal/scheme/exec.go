package scheme

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/macrat/ayd/internal/scheme/shell"
	"github.com/macrat/ayd/internal/scheme/textdecode"
	api "github.com/macrat/ayd/lib-ayd"
	"golang.org/x/crypto/ssh"
)

var (
	ErrMissingCommand = errors.New("missing command")
)

var (
	executeMessageRegex = regexp.MustCompile("(?m)^::([^:.\n\t]+)::[ \t]*([^\n]*)[ \t]*(?:\n|$)")
)

func getExecuteEnvByURL(u *api.URL) []string {
	env := os.Environ()
	for k, v := range u.ToURL().Query() {
		env = append(env, k+"="+v[len(v)-1])
	}
	return env
}

func parseExecMessage(message string, defaultStatus api.Status, defaultLatency time.Duration) (replacedMessage string, status api.Status, latency time.Duration, extra map[string]any) {
	status = defaultStatus
	latency = defaultLatency
	extra = make(map[string]any)

	ms := executeMessageRegex.FindAllStringSubmatch(message, -1)
	if ms == nil {
		return message, status, latency, extra
	}

	for _, m := range ms {
		switch m[1] {
		case "status":
			status = api.ParseStatus(strings.ToUpper(m[2]))
			message = strings.ReplaceAll(message, m[0], "")
		case "latency":
			if l, err := strconv.ParseFloat(m[2], 64); err == nil && l >= 0 {
				latency = time.Duration(l * float64(time.Millisecond))
				if latency < 0 {
					latency = time.Duration(math.MaxInt64)
				}
			}
			message = strings.ReplaceAll(message, m[0], "")
		case "time", "target", "message":
		default:
			var value any
			if json.Unmarshal([]byte(m[2]), &value) == nil {
				extra[m[1]] = value
			} else {
				extra[m[1]] = m[2]
			}
			message = strings.ReplaceAll(message, m[0], "")
		}
	}

	return strings.Trim(message, "\n"), status, latency, extra
}

func isUnknownExecutionError(err error) bool {
	if e := errors.Unwrap(err); e != nil {
		switch e.Error() {
		case "no such file or directory", "permission denied", "executable file not found in $PATH", "file does not exist", "executable file not found in %PATH%":
			return true
		}
	}
	return false
}

func runExternalCommand(ctx context.Context, f io.Writer, command string, args, env []string) (api.Status, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = env

	cmd.Stdout = f
	cmd.Stderr = f

	err := cmd.Run()

	status := api.StatusHealthy
	if err != nil {
		status = api.StatusFailure

		if isUnknownExecutionError(err) {
			status = api.StatusUnknown
		}
	}

	return status, err
}

func execResultToRecord(ctx context.Context, timestamp time.Time, target *api.URL, status api.Status, message string, latency time.Duration, extraOverride map[string]any, exitCode int, err error) api.Record {
	message = strings.Trim(message, "\n")

	if status != api.StatusHealthy && message == "" {
		message = err.Error()
	}

	var extra map[string]any
	message, status, latency, extra = parseExecMessage(message, status, latency)

	if err == nil {
		extra["exit_code"] = 0
	} else {
		if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
			// do not add exit_code if command cancelled by Ayd.
		} else if exitCode >= 0 {
			extra["exit_code"] = exitCode
		}
	}

	for k, v := range extraOverride {
		extra[k] = v
	}

	return timeoutOr(ctx, api.Record{
		Time:    timestamp,
		Target:  target,
		Status:  status,
		Message: message,
		Latency: latency,
		Extra:   extra,
	})
}

func NewExecScheme(u *api.URL) (Scheme, error) {
	if _, separator, variant := SplitScheme(u.Scheme); separator == '+' && variant == "ssh" {
		return NewExecSSHScheme(u)
	} else if separator == 0 {
		return NewExecLocalScheme(u)
	}

	return nil, ErrUnsupportedScheme
}

type ExecLocalScheme struct {
	target *api.URL
	env    []string
}

func NewExecLocalScheme(u *api.URL) (ExecLocalScheme, error) {
	s := ExecLocalScheme{}

	path := u.Opaque
	if u.Opaque == "" {
		path = u.Path
	}
	s.target = &api.URL{
		Scheme:   "exec",
		Opaque:   filepath.ToSlash(path),
		RawQuery: u.RawQuery,
		Fragment: u.Fragment,
	}

	if path == "" {
		return ExecLocalScheme{}, ErrMissingCommand
	}

	if _, err := exec.LookPath(filepath.FromSlash(path)); err != nil {
		return ExecLocalScheme{}, err
	}

	s.env = getExecuteEnvByURL(u)

	return s, nil
}

func (s ExecLocalScheme) Target() *api.URL {
	return s.target
}

func (s ExecLocalScheme) run(ctx context.Context, r Reporter, extraEnv []string) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	var args []string
	if s.target.Fragment != "" {
		args = []string{s.target.Fragment}
	}

	var buf bytes.Buffer

	stime := time.Now()
	status, err := runExternalCommand(
		ctx,
		&buf,
		filepath.FromSlash(s.target.Opaque),
		args,
		append(s.env, extraEnv...),
	)
	latency := time.Since(stime)

	output, e := textdecode.Bytes(buf.Bytes())
	if err == nil && e != nil {
		err = e
	}

	var exitErr *exec.ExitError
	exitCode := -1
	if errors.As(err, &exitErr) && exitErr.ExitCode() >= 0 {
		exitCode = exitErr.ExitCode()
	}

	r.Report(s.target, execResultToRecord(ctx, stime, s.target, status, output, latency, nil, exitCode, err))
}

func (s ExecLocalScheme) Probe(ctx context.Context, r Reporter) {
	s.run(ctx, r, nil)
}

func (s ExecLocalScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	env := []string{
		"ayd_time=" + lastRecord.Time.Format(time.RFC3339),
		"ayd_status=" + lastRecord.Status.String(),
		fmt.Sprintf("ayd_latency=%.3f", float64(lastRecord.Latency.Microseconds())/1000.0),
		"ayd_target=" + lastRecord.Target.String(),
		"ayd_message=" + lastRecord.Message,
		"ayd_extra={}",
	}

	if lastRecord.Extra != nil {
		if bs, err := json.Marshal(lastRecord.Extra); err == nil {
			env[len(env)-1] = "ayd_extra=" + string(bs)
		}
	}

	s.run(ctx, AlertReporter{s.target, r}, env)
}

type ExecSSHScheme struct {
	target *api.URL
	conf   sshConfig
	env    map[string]string
}

func NewExecSSHScheme(u *api.URL) (ExecSSHScheme, error) {
	if u.Path == "" {
		return ExecSSHScheme{}, ErrMissingCommand
	}

	q := u.ToURL().Query()
	if f := u.ToURL().Query().Get("fingerprint"); f != "" {
		q.Set("fingerprint", strings.ReplaceAll(f, " ", "+"))
	}
	u.RawQuery = q.Encode()

	conf, err := newSSHConfig(u)
	if err != nil {
		return ExecSSHScheme{}, err
	}

	env := make(map[string]string)
	for k, v := range u.ToURL().Query() {
		env[k] = v[len(v)-1]
	}

	return ExecSSHScheme{
		target: u,
		conf:   conf,
		env:    env,
	}, nil

}

func (s ExecSSHScheme) Target() *api.URL {
	return s.target
}

func (s ExecSSHScheme) run(ctx context.Context, r Reporter, extraEnv map[string]string) {
	timestamp := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	reportError := func(message string, err error, extra map[string]any) {
		r.Report(s.target, api.Record{
			Time:    timestamp,
			Target:  s.target,
			Status:  api.StatusUnknown,
			Message: fmt.Sprintf("%s: %s", message, err),
			Latency: time.Since(timestamp),
			Extra:   extra,
		})
	}

	conn, err := dialSSH(ctx, s.conf)
	if err != nil {
		reportError("failed to connect", err, conn.MakeExtra())
		return
	}
	defer conn.Close()

	sess, err := conn.Client.NewSession()
	if err != nil {
		reportError("failed to create a session", err, conn.MakeExtra())
		return
	}
	defer sess.Close()

	for k, v := range s.env {
		if k != "identityfile" && k != "fingerprint" {
			sess.Setenv(k, v)
		}
	}
	for k, v := range extraEnv {
		sess.Setenv(k, v)
	}

	command := shell.Escape(s.target.Path)
	if s.target.Fragment != "" {
		command += " " + shell.Escape(s.target.Fragment)
	}

	stime := time.Now()
	outputBytes, err := sess.CombinedOutput(command)
	latency := time.Since(stime)

	status := api.StatusHealthy
	if err != nil {
		status = api.StatusFailure

		var exitErr *ssh.ExitError
		if isUnknownExecutionError(err) || (errors.As(err, &exitErr) && exitErr.ExitStatus() == 126 || exitErr.ExitStatus() == 127) {
			status = api.StatusUnknown
		}
	}

	output, e := textdecode.UTF8(outputBytes)
	if err == nil && e != nil {
		err = e
	}

	var exitErr *ssh.ExitError
	exitCode := -1
	if errors.As(err, &exitErr) && exitErr.ExitStatus() >= 0 {
		exitCode = exitErr.ExitStatus()
	}

	r.Report(s.target, execResultToRecord(ctx, timestamp, s.target, status, output, latency, conn.MakeExtra(), exitCode, err))
}

func (s ExecSSHScheme) Probe(ctx context.Context, r Reporter) {
	s.run(ctx, r, nil)
}

func (s ExecSSHScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	env := map[string]string{
		"ayd_time":    lastRecord.Time.Format(time.RFC3339),
		"ayd_status":  lastRecord.Status.String(),
		"ayd_latency": fmt.Sprintf("%.3f", float64(lastRecord.Latency.Microseconds())/1000.0),
		"ayd_target":  lastRecord.Target.String(),
		"ayd_message": lastRecord.Message,
		"ayd_extra":   "{}",
	}

	if lastRecord.Extra != nil {
		if bs, err := json.Marshal(lastRecord.Extra); err == nil {
			env["ayd_extra"] = string(bs)
		}
	}

	s.run(ctx, AlertReporter{s.target, r}, env)
}
