package probe

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/store"
)

type ExecuteProbe struct {
	target *url.URL
	env    []string
}

func NewExecuteProbe(u *url.URL) (ExecuteProbe, error) {
	p := ExecuteProbe{}

	path := u.Opaque
	if u.Opaque == "" {
		path = u.Path
	}
	p.target = &url.URL{
		Scheme:   "exec",
		Opaque:   path,
		RawQuery: u.RawQuery,
		Fragment: u.Fragment,
	}

	p.env = os.Environ()
	for k, v := range u.Query() {
		p.env = append(p.env, k+"="+v[len(v)-1])
	}

	return p, nil
}

func (p ExecuteProbe) Target() *url.URL {
	return p.target
}

func getLatencyByMessage(message string, default_ time.Duration) (replacedMessage string, latency time.Duration) {
	latencyRe := regexp.MustCompile("(?m)^::latency::([0-9]+(?:\\.[0-9]+)?)(?:\n|$)")

	if m := latencyRe.FindAllStringSubmatch(message, -1); m != nil {
		if l, err := strconv.ParseFloat(m[len(m)-1][1], 64); err == nil {
			return strings.Trim(latencyRe.ReplaceAllString(message, ""), "\n"), time.Duration(l * float64(time.Millisecond))
		}
	}

	return message, default_
}

func getStatusByMessage(message string, default_ store.Status) (replacedMessage string, status store.Status) {
	statusRe := regexp.MustCompile("(?m)^::status::((?i:healthy|failure|aborted|unknown))(?:\n|$)")

	if m := statusRe.FindAllStringSubmatch(message, -1); m != nil {
		status = store.ParseStatus(strings.ToUpper(m[len(m)-1][1]))
		return strings.Trim(statusRe.ReplaceAllString(message, ""), "\n"), status
	}

	return message, default_
}

func ExecuteExternalCommand(ctx context.Context, r Reporter, target *url.URL, command, argument string, env []string) {
	var cmd *exec.Cmd
	if argument != "" {
		cmd = exec.CommandContext(ctx, command, argument)
	} else {
		cmd = exec.CommandContext(ctx, command)
	}

	cmd.Env = env

	st := time.Now()
	stdout, err := cmd.CombinedOutput()
	latency := time.Now().Sub(st)

	status := store.STATUS_HEALTHY
	message := strings.Trim(strings.ReplaceAll(strings.ReplaceAll(string(stdout), "\r\n", "\n"), "\r", "\n"), "\n")

	if err != nil {
		status = store.STATUS_FAILURE
		if e := errors.Unwrap(err); e != nil {
			switch e.Error() {
			case "no such file or directory", "permission denied", "executable file not found in $PATH", "file does not exist", "executable file not found in %PATH%":
				status = store.STATUS_UNKNOWN
			}
		}

		if message == "" {
			message = err.Error()
		}
	}

	message, latency = getLatencyByMessage(message, latency)
	message, status = getStatusByMessage(message, status)

	r.Report(timeoutOr(ctx, store.Record{
		CheckedAt: st,
		Target:    target,
		Status:    status,
		Message:   message,
		Latency:   latency,
	}))
}

func (p ExecuteProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	ExecuteExternalCommand(ctx, r, p.target, p.target.Opaque, p.target.Fragment, p.env)
}
