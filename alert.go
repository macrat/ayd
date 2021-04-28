package main

import (
	"context"
	"net/url"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

type Alert struct {
	target *url.URL
}

func NewAlert(target string) (*Alert, error) {
	p, err := probe.New(target)
	if err != nil {
		return nil, err
	}

	return &Alert{p.Target()}, nil
}

func (a *Alert) Target() *url.URL {
	return &url.URL{
		Scheme: "alert",
		Opaque: a.target.String(),
	}
}

func (a *Alert) Trigger(ctx context.Context, incident *store.Incident, r probe.Reporter) {
	qs := a.target.Query()
	qs.Set("ayd_target", incident.Target.String())
	qs.Set("ayd_checked_at", incident.CausedAt.Format(time.RFC3339))
	qs.Set("ayd_status", incident.Status.String())

	u := *a.target
	u.RawQuery = qs.Encode()

	p, err := probe.NewFromURL(&u)
	if err != nil {
		r.Report(store.Record{
			CheckedAt: time.Now(),
			Target:    a.Target(),
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
