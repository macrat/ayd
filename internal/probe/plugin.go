package probe

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type PluginProbe struct {
	target  *url.URL
	command string
}

func NewPluginProbe(u *url.URL) (PluginProbe, error) {
	scheme := strings.SplitN(u.Scheme, "-", 2)[0]
	scheme = strings.SplitN(scheme, "+", 2)[0]

	if scheme == "ayd" || scheme == "alert" {
		return PluginProbe{}, ErrUnsupportedScheme
	}

	p := PluginProbe{
		target:  u,
		command: "ayd-" + scheme + "-probe",
	}

	if _, err := exec.LookPath(p.command); errors.Is(err, exec.ErrNotFound) {
		return PluginProbe{}, ErrUnsupportedScheme
	} else if err != nil {
		return PluginProbe{}, err
	}

	return p, nil
}

func (p PluginProbe) Target() *url.URL {
	return p.target
}

func ExecutePlugin(ctx context.Context, r Reporter, scope string, target *url.URL, command string, args, env []string) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	stime := time.Now()
	output, status, err := runExternalCommand(ctx, command, args, env)
	latency := time.Now().Sub(stime)

	count := 0

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		rec, err := api.ParseRecord(text)
		if err == nil {
			count++
			r.Report(rec)
			continue
		}

		r.Report(api.Record{
			CheckedAt: time.Now(),
			Target:    &url.URL{Scheme: "ayd", Opaque: scope + ":plugin:" + target.String()},
			Status:    api.StatusUnknown,
			Message:   fmt.Sprintf("%s: %#v", err, text),
			Latency:   latency,
		})
	}

	if err != nil || count == 0 {
		msg := ""
		if err != nil {
			msg = err.Error()
		}

		r.Report(timeoutOr(ctx, api.Record{
			CheckedAt: stime,
			Target:    target,
			Status:    status,
			Message:   msg,
			Latency:   latency,
		}))
	}
}

func (p PluginProbe) Check(ctx context.Context, r Reporter) {
	ExecutePlugin(ctx, r, "probe", p.target, p.command, []string{p.target.String()}, os.Environ())
}
