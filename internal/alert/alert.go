package alert

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/probe"
	api "github.com/macrat/ayd/lib-ayd"
)

type Alert interface {
	Trigger(context.Context, *api.Incident, probe.Reporter)
}

func New(target string) (Alert, error) {
	p, err := probe.WithoutPlugin(probe.New(target))
	if err == probe.ErrUnsupportedScheme {
		return NewPluginAlert(target)
	} else if err != nil {
		return nil, err
	}

	return ProbeAlert{p.Target()}, nil
}

type ReplaceReporter struct {
	Target   *url.URL
	Upstream probe.Reporter
}

func (r ReplaceReporter) Report(rec api.Record) {
	scheme := strings.SplitN(rec.Target.Scheme, "-", 2)[0]
	scheme = strings.SplitN(scheme, "+", 2)[0]

	if scheme != "alert" && scheme != "ayd" {
		rec.Target = r.Target
	}
	r.Upstream.Report(rec)
}

type ProbeAlert struct {
	target *url.URL
}

func (a ProbeAlert) Trigger(ctx context.Context, incident *api.Incident, r probe.Reporter) {
	qs := a.target.Query()
	qs.Set("ayd_caused_at", incident.CausedAt.Format(time.RFC3339))
	qs.Set("ayd_status", incident.Status.String())
	qs.Set("ayd_target", incident.Target.String())
	qs.Set("ayd_message", incident.Message)

	u := *a.target
	u.RawQuery = qs.Encode()

	reporter := ReplaceReporter{
		&url.URL{Scheme: "alert", Opaque: a.target.String()},
		r,
	}

	p, err := probe.WithoutPlugin(probe.NewFromURL(&u))
	if err != nil {
		reporter.Report(api.Record{
			CheckedAt: time.Now(),
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
		return
	}

	p.Check(ctx, reporter)
}

type AlertReporter struct {
	Upstream probe.Reporter
}

func (r AlertReporter) Report(rec api.Record) {
	scheme := strings.SplitN(rec.Target.Scheme, "-", 2)[0]
	scheme = strings.SplitN(scheme, "+", 2)[0]

	if scheme != "alert" && scheme != "ayd" {
		rec.Target = &url.URL{
			Scheme: "alert",
			Opaque: rec.Target.String(),
		}
	}
	r.Upstream.Report(rec)
}

type PluginAlert struct {
	target  *url.URL
	command string
}

func NewPluginAlert(target string) (PluginAlert, error) {
	u, err := url.Parse(target)
	if err != nil {
		return PluginAlert{}, err
	}

	scheme := strings.SplitN(u.Scheme, "-", 2)[0]
	scheme = strings.SplitN(scheme, "+", 2)[0]

	if scheme == "ayd" || scheme == "alert" {
		return PluginAlert{}, probe.ErrUnsupportedScheme
	}

	p := PluginAlert{
		target:  u,
		command: "ayd-" + scheme + "-alert",
	}

	if _, err := exec.LookPath(p.command); errors.Unwrap(err) == exec.ErrNotFound {
		return PluginAlert{}, probe.ErrUnsupportedScheme
	} else if err != nil {
		return PluginAlert{}, err
	}

	return p, nil
}

func (a PluginAlert) Trigger(ctx context.Context, incident *api.Incident, r probe.Reporter) {
	probe.ExecutePlugin(
		ctx,
		AlertReporter{r},
		"alert",
		a.target,
		a.command,
		[]string{
			a.target.String(),
			incident.CausedAt.Format(time.RFC3339),
			incident.Status.String(),
			incident.Target.String(),
			incident.Message,
		},
		os.Environ(),
	)
}
