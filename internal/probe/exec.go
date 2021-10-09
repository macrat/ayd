package probe

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrMissingCommand = errors.New("missing command")
)

var (
	executeLatencyRe = regexp.MustCompile("(?m)^::latency::([0-9]+(?:\\.[0-9]+)?)(?:\n|$)")
	executeStatusRe  = regexp.MustCompile("(?m)^::status::((?i:healthy|failure|aborted|unknown))(?:\n|$)")
)

func getExecuteEnvByURL(u *url.URL) []string {
	env := os.Environ()
	for k, v := range u.Query() {
		env = append(env, k+"="+v[len(v)-1])
	}
	return env
}

type ExecuteProbe struct {
	target *url.URL
	env    []string
}

func NewExecuteProbe(u *url.URL) (ExecuteProbe, error) {
	p := ExecuteProbe{}

	if _, separator, _ := SplitScheme(u.Scheme); separator != 0 {
		return ExecuteProbe{}, ErrUnsupportedScheme
	}

	path := u.Opaque
	if u.Opaque == "" {
		path = u.Path
	}
	p.target = &url.URL{
		Scheme:   "exec",
		Opaque:   filepath.ToSlash(path),
		RawQuery: u.RawQuery,
		Fragment: u.Fragment,
	}

	if path == "" {
		return ExecuteProbe{}, ErrMissingCommand
	}

	if _, err := exec.LookPath(filepath.FromSlash(path)); err != nil {
		return ExecuteProbe{}, err
	}

	p.env = getExecuteEnvByURL(u)

	return p, nil
}

func (p ExecuteProbe) Target() *url.URL {
	return p.target
}

func getLatencyByMessage(message string, default_ time.Duration) (replacedMessage string, latency time.Duration) {
	if m := executeLatencyRe.FindAllStringSubmatch(message, -1); m != nil {
		if l, err := strconv.ParseFloat(m[len(m)-1][1], 64); err == nil {
			return strings.Trim(executeLatencyRe.ReplaceAllString(message, ""), "\n"), time.Duration(l * float64(time.Millisecond))
		}
	}

	return message, default_
}

func getStatusByMessage(message string, default_ api.Status) (replacedMessage string, status api.Status) {
	if m := executeStatusRe.FindAllStringSubmatch(message, -1); m != nil {
		var status api.Status
		status.UnmarshalText([]byte(strings.ToUpper(m[len(m)-1][1])))
		return strings.Trim(executeStatusRe.ReplaceAllString(message, ""), "\n"), status
	}

	return message, default_
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

	output = autoDecode(buf.Bytes())

	status = api.StatusHealthy
	if err != nil {
		status = api.StatusFailure

		if isUnknownExecutionError(err) {
			status = api.StatusUnknown
		}
	}

	return
}

func (p ExecuteProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	var args []string
	if p.target.Fragment != "" {
		args = []string{p.target.Fragment}
	}

	stime := time.Now()
	output, status, err := runExternalCommand(ctx, filepath.FromSlash(p.target.Opaque), args, p.env)
	latency := time.Now().Sub(stime)

	message := strings.Trim(output, "\n")

	if status != api.StatusHealthy && message == "" {
		message = err.Error()
	}

	message, latency = getLatencyByMessage(message, latency)
	message, status = getStatusByMessage(message, status)

	r.Report(timeoutOr(ctx, api.Record{
		CheckedAt: stime,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   latency,
	}))
}
