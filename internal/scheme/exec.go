package scheme

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/scheme/textdecode"
	api "github.com/macrat/ayd/lib-ayd"
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

type ExecScheme struct {
	target *api.URL
	env    []string
}

func NewExecScheme(u *api.URL) (ExecScheme, error) {
	s := ExecScheme{}

	if _, separator, _ := SplitScheme(u.Scheme); separator != 0 {
		return ExecScheme{}, ErrUnsupportedScheme
	}

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
		return ExecScheme{}, ErrMissingCommand
	}

	if _, err := exec.LookPath(filepath.FromSlash(path)); err != nil {
		return ExecScheme{}, err
	}

	s.env = getExecuteEnvByURL(u)

	return s, nil
}

func (s ExecScheme) Target() *api.URL {
	return s.target
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

func runExternalCommand(ctx context.Context, command string, args, env []string) (output string, status api.Status, err error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = env

	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf

	err = cmd.Run()

	status = api.StatusHealthy
	if err != nil {
		status = api.StatusFailure

		if isUnknownExecutionError(err) {
			status = api.StatusUnknown
		}
	}

	var e error
	output, e = textdecode.Bytes(buf.Bytes())
	if err == nil {
		err = e
	}

	return
}

func (s ExecScheme) run(ctx context.Context, r Reporter, extraEnv []string) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	var args []string
	if s.target.Fragment != "" {
		args = []string{s.target.Fragment}
	}

	stime := time.Now()
	output, status, err := runExternalCommand(
		ctx,
		filepath.FromSlash(s.target.Opaque),
		args,
		append(s.env, extraEnv...),
	)
	latency := time.Since(stime)

	message := strings.Trim(output, "\n")

	if status != api.StatusHealthy && message == "" {
		message = err.Error()
	}

	var extra map[string]any
	message, status, latency, extra = parseExecMessage(message, status, latency)

	if err == nil {
		extra["exit_code"] = 0
	} else {
		var exitErr *exec.ExitError
		if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
			// do not add exit_code if command cancelled by Ayd.
		} else if errors.As(err, &exitErr) && exitErr.ExitCode() >= 0 {
			extra["exit_code"] = exitErr.ExitCode()
		}
	}

	r.Report(s.target, timeoutOr(ctx, api.Record{
		Time:    stime,
		Target:  s.target,
		Status:  status,
		Message: message,
		Latency: latency,
		Extra:   extra,
	}))
}

func (s ExecScheme) Probe(ctx context.Context, r Reporter) {
	s.run(ctx, r, nil)
}

func (s ExecScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
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
