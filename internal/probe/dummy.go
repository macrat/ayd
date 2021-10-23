package probe

import (
	"context"
	"errors"
	"math/rand"
	"net/url"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type DummyProbe struct {
	target  *url.URL
	random  bool
	status  api.Status
	latency time.Duration
	message string
}

func NewDummyProbe(u *url.URL) (DummyProbe, error) {
	p := DummyProbe{target: &url.URL{Scheme: u.Scheme, Opaque: u.Opaque, Fragment: u.Fragment}}
	if u.Opaque == "" {
		p.target.Opaque = u.Host
	}

	p.target.Opaque = strings.ToLower(p.target.Opaque)
	switch p.target.Opaque {
	case "", "healthy":
		p.status = api.StatusHealthy
	case "debased":
		p.status = api.StatusDebased
	case "failure":
		p.status = api.StatusFailure
	case "aborted":
		p.status = api.StatusAborted
	case "unknown":
		p.status = api.StatusUnknown
	case "random":
		p.random = true
	default:
		return DummyProbe{}, errors.New("opaque must healthy, debased, failure, aborted, unknown, or random")
	}

	query := url.Values{}

	if latency := u.Query().Get("latency"); latency != "" {
		d, err := time.ParseDuration(latency)
		if err != nil {
			return DummyProbe{}, err
		}
		p.latency = d
		query.Set("latency", d.String())
	}

	if message := u.Query().Get("message"); message != "" {
		p.message = message
		query.Set("message", message)
	}

	p.target.RawQuery = query.Encode()

	return p, nil
}

func (p DummyProbe) Status() api.Status {
	if !p.random {
		return p.status
	}

	return []api.Status{
		api.StatusHealthy,
		api.StatusDebased,
		api.StatusFailure,
		api.StatusUnknown,
	}[rand.Intn(4)]
}

func (p DummyProbe) Target() *url.URL {
	return p.target
}

func (p DummyProbe) Check(ctx context.Context, r Reporter) {
	stime := time.Now()

	latency := p.latency
	if p.target.Query().Get("latency") == "" {
		latency = time.Duration(rand.Intn(10000)) * time.Microsecond
	}

	rec := api.Record{
		CheckedAt: stime,
		Status:    p.Status(),
		Target:    p.target,
		Latency:   latency,
		Message:   p.message,
	}

	select {
	case <-time.After(latency):
	case <-ctx.Done():
		rec.Latency = time.Now().Sub(stime)
	}

	r.Report(timeoutOr(ctx, rec))
}
