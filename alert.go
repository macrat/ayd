package main

import (
	"context"
	"errors"
	"fmt"
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

func NewAlert(target, externalURL string) (Alert, error) {
	p, err := probe.WithoutPlugin(probe.New(target))
	if err == probe.ErrUnsupportedScheme {
		return NewPluginAlert(target, externalURL)
	} else if err != nil {
		return nil, err
	}

	return &ProbeAlert{p.Target(), externalURL}, nil
}

type ProbeAlert struct {
	target   *url.URL
	external string
}

func (a ProbeAlert) Trigger(ctx context.Context, incident *store.Incident, r probe.Reporter) {
	qs := a.target.Query()
	qs.Set("ayd_url", a.external)
	qs.Set("ayd_target", incident.Target.String())
	qs.Set("ayd_checked_at", incident.CausedAt.Format(time.RFC3339))
	qs.Set("ayd_status", incident.Status.String())

	u := *a.target
	u.RawQuery = qs.Encode()

	p, err := probe.WithoutPlugin(probe.NewFromURL(&u))
	if err != nil {
		r.Report(store.Record{
			CheckedAt: time.Now(),
			Target:    &url.URL{Scheme: "alert", Opaque: a.target.String()},
			Status:    store.STATUS_UNKNOWN,
			Message:   err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(ctx, TASK_TIMEOUT)
	defer cancel()

	p.Check(ctx, AlertReporter{r})
}

type AlertReporter struct {
	Upstream probe.Reporter
}

func (r AlertReporter) Report(rec store.Record) {
	rec.Target = &url.URL{
		Scheme: "alert",
		Opaque: rec.Target.String(),
	}
	r.Upstream.Report(rec)
}

type PluginAlert struct {
	target  *url.URL
	command string
	env     []string
}

func NewPluginAlert(target, externalURL string) (PluginAlert, error) {
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
		env:     os.Environ(),
	}

	if _, err := exec.LookPath(p.command); errors.Unwrap(err) == exec.ErrNotFound {
		return PluginAlert{}, probe.ErrUnsupportedScheme
	} else if err != nil {
		return PluginAlert{}, err
	}

	p.env = append(
		p.env,
		fmt.Sprintf("ayd_url=%s", externalURL),
	)

	return p, nil
}

func (a PluginAlert) Trigger(ctx context.Context, incident *store.Incident, r probe.Reporter) {
	ctx, cancel := context.WithTimeout(ctx, TASK_TIMEOUT)
	defer cancel()

	probe.ExecuteExternalCommand(
		ctx,
		r,
		incident.Target,
		a.command,
		"",
		append(
			a.env,
			fmt.Sprintf("ayd_target=%s", incident.Target.String()),
			fmt.Sprintf("ayd_checked_at=%s", incident.CausedAt.Format(time.RFC3339)),
			fmt.Sprintf("ayd_status=%s", incident.Status.String()),
		),
	)
}
