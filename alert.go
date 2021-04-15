package main

import (
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

func (a *Alert) Trigger(r store.Record) []store.Record {
	qs := a.target.Query()
	qs.Set("ayd_target", r.Target.String())
	qs.Set("ayd_checked_at", r.CheckedAt.Format(time.RFC3339))
	qs.Set("ayd_status", r.Status.String())

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

	result := p.Check()
	for i := range result {
		result[i].Target = a.Target()
	}

	return result
}

func (a *Alert) TriggerIfNeed(rs []store.Record) []store.Record {
	if a == nil {
		return nil
	}

	var result []store.Record
	for _, r := range rs {
		if r.Status != store.STATUS_HEALTHY {
			result = append(result, a.Trigger(r)...)
		}
	}
	return result
}
