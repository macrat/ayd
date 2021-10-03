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
	p := DummyProbe{target: &url.URL{Scheme: "dummy", Opaque: u.Opaque, Fragment: u.Fragment}}
	if u.Opaque == "" {
		p.target.Opaque = u.Host
	}

	p.target.Opaque = strings.ToLower(p.target.Opaque)
	switch p.target.Opaque {
	case "", "healthy":
		p.status = api.StatusHealthy
	case "failure":
		p.status = api.StatusFailure
	case "aborted":
		p.status = api.StatusAborted
	case "unknown":
		p.status = api.StatusUnknown
	case "random":
		p.random = true
	default:
		return DummyProbe{}, errors.New("opaque must healthy, failure, aborted, unknown, or random")
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
		api.StatusUnknown,
		api.StatusFailure,
	}[rand.Intn(3)]
}

func (p DummyProbe) Target() *url.URL {
	return p.target
}

func (p DummyProbe) Check(ctx context.Context, r Reporter) {
	stime := time.Now()

	rec := api.Record{
		CheckedAt: stime,
		Status:    p.Status(),
		Target:    p.target,
		Latency:   p.latency,
		Message:   p.message,
	}

	if p.latency > 0 {
		select {
		case <-time.After(p.latency):
		case <-ctx.Done():
			rec.Latency = time.Now().Sub(stime)
		}
	} else {
		rec.Latency = time.Now().Sub(stime)
	}

	r.Report(timeoutOr(ctx, rec))
}