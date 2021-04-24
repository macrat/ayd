package probe

import (
	"context"
	"errors"
	"fmt"
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
		p.env = append(p.env, fmt.Sprintf("%s=%s", k, v[len(v)-1]))
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
	statusRe := regexp.MustCompile("(?m)^::status::((?i:healthy|failure|unknown))(?:\n|$)")

	if m := statusRe.FindAllStringSubmatch(message, -1); m != nil {
		switch strings.ToLower(m[len(m)-1][1]) {
		case "healthy":
			status = store.STATUS_HEALTHY
		case "failure":
			status = store.STATUS_FAILURE
		case "unknown":
			status = store.STATUS_UNKNOWN
		}
		return strings.Trim(statusRe.ReplaceAllString(message, ""), "\n"), status
	}

	return message, default_
}

func (p ExecuteProbe) Check(ctx context.Context) []store.Record {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if p.target.Fragment != "" {
		cmd = exec.CommandContext(ctx, p.target.Opaque, p.target.Fragment)
	} else {
		cmd = exec.CommandContext(ctx, p.target.Opaque)
	}

	cmd.Env = p.env

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

		select {
		case <-ctx.Done():
			status = store.STATUS_UNKNOWN
			message = "timeout"
		default:
		}

		if message == "" {
			message = err.Error()
		}
	}

	message, latency = getLatencyByMessage(message, latency)
	message, status = getStatusByMessage(message, status)

	return []store.Record{{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   latency,
	}}
}
