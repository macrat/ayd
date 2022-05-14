package scheme

import (
	"context"
	"errors"
	"math/rand"
	"net/url"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type DummyScheme struct {
	target  *api.URL
	random  bool
	status  api.Status
	latency time.Duration
	message string
}

func NewDummyScheme(u *api.URL) (DummyScheme, error) {
	s := DummyScheme{target: &api.URL{Scheme: u.Scheme, Opaque: u.Opaque, Fragment: u.Fragment}}
	if u.Opaque == "" {
		s.target.Opaque = u.Host
	}

	s.target.Opaque = strings.ToLower(s.target.Opaque)
	switch s.target.Opaque {
	case "", "healthy":
		s.status = api.StatusHealthy
	case "degrade":
		s.status = api.StatusDegrade
	case "failure":
		s.status = api.StatusFailure
	case "aborted":
		s.status = api.StatusAborted
	case "unknown":
		s.status = api.StatusUnknown
	case "random":
		s.random = true
	default:
		return DummyScheme{}, errors.New("opaque must healthy, degrade, failure, aborted, unknown, or random")
	}

	query := url.Values{}

	if latency := u.ToURL().Query().Get("latency"); latency != "" {
		d, err := time.ParseDuration(latency)
		if err != nil {
			return DummyScheme{}, err
		}
		s.latency = d
		query.Set("latency", d.String())
	}

	if message := u.ToURL().Query().Get("message"); message != "" {
		s.message = message
		query.Set("message", message)
	}

	s.target.RawQuery = query.Encode()

	return s, nil
}

func (s DummyScheme) Status() api.Status {
	if !s.random {
		return s.status
	}

	return []api.Status{
		api.StatusHealthy,
		api.StatusDegrade,
		api.StatusFailure,
		api.StatusUnknown,
	}[rand.Intn(4)]
}

func (s DummyScheme) Target() *api.URL {
	return s.target
}

func (s DummyScheme) Probe(ctx context.Context, r Reporter) {
	stime := time.Now()

	latency := s.latency
	if s.target.ToURL().Query().Get("latency") == "" {
		latency = time.Duration(rand.Intn(10000)) * time.Microsecond
	}

	rec := api.Record{
		CheckedAt: stime,
		Status:    s.Status(),
		Target:    s.target,
		Latency:   latency,
		Message:   s.message,
	}

	select {
	case <-time.After(latency):
	case <-ctx.Done():
		rec.Latency = time.Since(stime)
	}

	r.Report(s.target, timeoutOr(ctx, rec))
}

func (s DummyScheme) Alert(ctx context.Context, r Reporter, _ api.Record) {
	s.Probe(ctx, AlertReporter{s.target, r})
}
