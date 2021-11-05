package scheme

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
	tracker *TargetTracker
}

// PluginCandidates makes scheme name candidates of plugin by URL scheme.
func PluginCandidates(scheme string) []string {
	var xs []string

	for i, x := range scheme {
		if x == '-' || x == '+' {
			xs = append(xs, scheme[:i])
		}
	}

	xs = append(xs, scheme)

	return xs
}

// FindPlugin finds a plugin for URL scheme.
// It choice the longest name plugin.
func FindPlugin(scheme, scope string) (commandName string, err error) {
	candidates := PluginCandidates(scheme)
	for i := range candidates {
		commandName = "ayd-" + candidates[len(candidates)-i-1] + "-" + scope
		_, err = exec.LookPath(commandName)
		if err == nil || !errors.Is(err, exec.ErrNotFound) {
			return
		}
	}
	return "", ErrUnsupportedScheme
}

func NewPluginProbe(u *url.URL) (PluginProbe, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	if scheme == "ayd" || scheme == "alert" {
		return PluginProbe{}, ErrUnsupportedScheme
	}

	p := PluginProbe{
		target:  u,
		tracker: &TargetTracker{},
	}
	p.tracker.Activate(u)

	if _, err := FindPlugin(u.Scheme, "probe"); err != nil {
		return PluginProbe{}, err
	}

	return p, nil
}

func (p PluginProbe) Target() *url.URL {
	return p.target
}

func ExecutePlugin(ctx context.Context, r Reporter, scope string, target *url.URL, args, env []string) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	command, err := FindPlugin(target.Scheme, scope)
	if err != nil {
		r.Report(target, api.Record{
			CheckedAt: time.Now(),
			Target:    target,
			Status:    api.StatusUnknown,
			Message:   scope + " plugin for " + target.Scheme + " was not found",
		})
		return
	}

	stime := time.Now()
	output, status, err := runExternalCommand(ctx, command, args, env)
	latency := time.Now().Sub(stime)

	count := 0

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		rec, err := api.ParseRecord(text)
		if err == nil {
			count++
			r.Report(target, rec)
			continue
		}

		r.Report(target, api.Record{
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

		r.Report(target, timeoutOr(ctx, api.Record{
			CheckedAt: stime,
			Target:    target,
			Status:    status,
			Message:   msg,
			Latency:   latency,
		}))
	}
}

func (p PluginProbe) Probe(ctx context.Context, r Reporter) {
	r = p.tracker.PrepareReporter(p.target, r)
	ExecutePlugin(ctx, r, "probe", p.target, []string{p.target.String()}, os.Environ())

	r.DeactivateTarget(p.target, p.tracker.Inactives()...)
}
