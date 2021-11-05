package scheme

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

// PluginScheme is the plugin handler. This implements both of Prober interface and Alerter interface.
type PluginScheme struct {
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

func NewPluginScheme(u *url.URL, scope string) (PluginScheme, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	if scheme == "ayd" || scheme == "alert" {
		return PluginScheme{}, ErrUnsupportedScheme
	}

	p := PluginScheme{
		target:  u,
		tracker: &TargetTracker{},
	}
	p.tracker.Activate(u)

	if _, err := FindPlugin(u.Scheme, scope); err != nil {
		return PluginScheme{}, err
	}

	return p, nil
}

func NewPluginProbe(u *url.URL) (PluginScheme, error) {
	return NewPluginScheme(u, "probe")
}

func NewPluginAlert(u *url.URL) (PluginScheme, error) {
	return NewPluginScheme(u, "alert")
}

func (p PluginScheme) Target() *url.URL {
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

func (p PluginScheme) Probe(ctx context.Context, r Reporter) {
	r = p.tracker.PrepareReporter(p.target, r)
	ExecutePlugin(ctx, r, "probe", p.target, []string{p.target.String()}, os.Environ())
	r.DeactivateTarget(p.target, p.tracker.Inactives()...)
}

func (p PluginScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	ExecutePlugin(
		ctx,
		AlertReporter{&url.URL{Scheme: "alert", Opaque: p.target.String()}, r},
		"alert",
		p.target,
		[]string{
			p.target.String(),
			lastRecord.CheckedAt.Format(time.RFC3339),
			lastRecord.Status.String(),
			strconv.FormatFloat(float64(lastRecord.Latency.Microseconds())/1000.0, 'f', -1, 64),
			lastRecord.Target.String(),
			lastRecord.Message,
		},
		os.Environ(),
	)
}
