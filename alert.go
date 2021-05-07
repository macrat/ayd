package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

type Alert interface {
	Trigger(context.Context, *store.Incident, probe.Reporter)
}

func NewAlert(target string) (Alert, error) {
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

func (r ReplaceReporter) Report(rec store.Record) {
	rec.Target = r.Target
	r.Upstream.Report(rec)
}

type ProbeAlert struct {
	target *url.URL
}

func (a ProbeAlert) Trigger(ctx context.Context, incident *store.Incident, r probe.Reporter) {
	qs := a.target.Query()
	qs.Set("ayd_target", incident.Target.String())
	qs.Set("ayd_checked_at", incident.CausedAt.Format(time.RFC3339))
	qs.Set("ayd_status", incident.Status.String())

	u := *a.target
	u.RawQuery = qs.Encode()

	reporter := ReplaceReporter{
		&url.URL{Scheme: "alert", Opaque: a.target.String()},
		r,
	}

	p, err := probe.WithoutPlugin(probe.NewFromURL(&u))
	if err != nil {
		reporter.Report(store.Record{
			CheckedAt: time.Now(),
			Status:    store.STATUS_UNKNOWN,
			Message:   err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(ctx, TASK_TIMEOUT)
	defer cancel()

	p.Check(ctx, reporter)
}

type AlertReporter struct {
	Upstream probe.Reporter
}

func (r AlertReporter) Report(rec store.Record) {
	if rec.Target.Scheme != "alert" {
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

	if u.Scheme == "ayd" || u.Scheme == "alert" {
		return PluginAlert{}, probe.ErrUnsupportedScheme
	}

	p := PluginAlert{
		target:  u,
		command: "ayd-" + u.Scheme + "-alert",
	}

	if _, err := exec.LookPath(p.command); errors.Unwrap(err) == exec.ErrNotFound {
		return PluginAlert{}, probe.ErrUnsupportedScheme
	} else if err != nil {
		return PluginAlert{}, err
	}

	return p, nil
}

func (a PluginAlert) Trigger(ctx context.Context, incident *store.Incident, r probe.Reporter) {
	ctx, cancel := context.WithTimeout(ctx, TASK_TIMEOUT)
	defer cancel()

	probe.ExecutePlugin(
		ctx,
		AlertReporter{r},
		a.target,
		a.command,
		[]string{
			a.target.String(),
			incident.Target.String(),
			incident.Status.String(),
			incident.CausedAt.Format(time.RFC3339),
		},
		os.Environ(),
	)
}
