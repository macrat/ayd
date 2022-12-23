package scheme

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/scheme/textdecode"
	api "github.com/macrat/ayd/lib-ayd"
)

// PluginScheme is the plugin handler. This implements both of Prober interface and Alerter interface.
type PluginScheme struct {
	target  *api.URL
	tracker *TargetTracker
}

// pluginCandidates makes scheme name candidates of plugin by URL scheme.
// The output is priority ascending order, which means the first candidate has the lowest priority.
func pluginCandidates(scheme, scope string) []string {
	var xs []string

	for i, x := range scheme {
		if x == '-' || x == '+' {
			xs = append(
				xs,
				"ayd-"+scheme[:i]+"-scheme",
				"ayd-"+scheme[:i]+"-"+scope,
			)
		}
	}

	xs = append(
		xs,
		"ayd-"+scheme+"-scheme",
		"ayd-"+scheme+"-"+scope,
	)

	return xs
}

// findPlugin finds a plugin for URL scheme.
// It choice the longest name plugin.
func findPlugin(scheme, scope string) (commandName string, err error) {
	candidates := pluginCandidates(scheme, scope)
	for i := range candidates {
		commandName = candidates[len(candidates)-i-1]
		_, err = exec.LookPath(commandName)
		if err == nil || !errors.Is(err, exec.ErrNotFound) {
			return
		}
	}
	return "", ErrUnsupportedScheme
}

func NewPluginScheme(u *api.URL, scope string) (PluginScheme, error) {
	scheme, _, _ := SplitScheme(u.Scheme)

	if scheme == "ayd" || scheme == "alert" {
		return PluginScheme{}, ErrUnsupportedScheme
	}

	p := PluginScheme{
		target:  u,
		tracker: &TargetTracker{},
	}
	p.tracker.Activate(u)

	if _, err := findPlugin(u.Scheme, scope); err != nil {
		return PluginScheme{}, err
	}

	return p, nil
}

func NewPluginProbe(u *api.URL) (PluginScheme, error) {
	return NewPluginScheme(u, "probe")
}

func NewPluginAlert(u *api.URL) (PluginScheme, error) {
	return NewPluginScheme(u, "alert")
}

func (p PluginScheme) Target() *api.URL {
	return p.target
}

func (p PluginScheme) execute(ctx context.Context, r Reporter, scope string, args []string) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	command, err := findPlugin(p.target.Scheme, scope)
	if err != nil {
		r.Report(p.target, api.Record{
			Time:    time.Now(),
			Target:  p.target,
			Status:  api.StatusUnknown,
			Message: scope + " plugin for " + p.target.Scheme + " was not found",
		})
		return
	}

	rb, wb := io.Pipe()
	defer rb.Close()

	stime := time.Now()
	var status api.Status
	var latency time.Duration
	go func() {
		status, err = runExternalCommand(ctx, wb, command, args, os.Environ())
		latency = time.Since(stime)
		wb.Close()
	}()

	count := 0

	var invalidLines []string

	scanner := bufio.NewScanner(rb)
	for scanner.Scan() {
		text, err := textdecode.Bytes(scanner.Bytes())
		if err != nil {
			invalidLines = append(invalidLines, scanner.Text())
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		rec, err := api.ParseRecord(text)
		if err != nil {
			invalidLines = append(invalidLines, scanner.Text())
			continue
		}

		rec.Time = rec.Time.Local()

		count++
		r.Report(p.target, rec)
	}

	if invalidLines != nil {
		r.Report(p.target, api.Record{
			Time:    stime,
			Target:  p.target,
			Status:  api.StatusUnknown,
			Message: "the plugin reported invalid records",
			Latency: latency,
			Extra: map[string]any{
				"raw_message": strings.Join(invalidLines, "\n"),
			},
		})
	}

	if err != nil || (invalidLines == nil && count == 0) {
		msg := ""
		if err != nil {
			msg = err.Error()
		}

		r.Report(p.target, timeoutOr(ctx, api.Record{
			Time:    stime,
			Target:  p.target,
			Status:  status,
			Message: msg,
			Latency: latency,
		}))
	}
}

func (p PluginScheme) Probe(ctx context.Context, r Reporter) {
	r = p.tracker.PrepareReporter(p.target, r)
	p.execute(ctx, r, "probe", []string{p.target.String()})
	r.DeactivateTarget(p.target, p.tracker.Inactives()...)
}

func (p PluginScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	args := []string{
		p.target.String(),
		lastRecord.String(),
	}

	p.execute(
		ctx,
		AlertReporter{&api.URL{Scheme: "alert", Opaque: p.target.String()}, r},
		"alert",
		args,
	)
}
