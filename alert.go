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
	p, err := probe.Get(target)
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

func (a *Alert) Trigger(incident *store.Incident) []store.Record {
	qs := a.target.Query()
	qs.Set("ayd_target", incident.Target.String())
	qs.Set("ayd_checked_at", incident.CausedAt.Format(time.RFC3339))
	qs.Set("ayd_status", incident.Status.String())

	u := *a.target
	u.RawQuery = qs.Encode()

	p, err := probe.GetByURL(&u)
	if err != nil {
		return []store.Record{{
			CheckedAt: time.Now(),
			Target:    a.Target(),
			Status:    store.STATUS_UNKNOWN,
			Message:   err.Error(),
		}}
	}

	ctx, cancel := context.WithTimeout(context.Background(), TASK_TIMEOUT)
	defer cancel()

	result := p.Check(ctx)
	for i := range result {
		result[i].Target = &url.URL{
			Scheme: "alert",
			Opaque: result[i].Target.String(),
		}
	}

	return result
}
