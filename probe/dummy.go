package probe

import (
	"context"
	"errors"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/macrat/ayd/store"
)

type DummyProbe struct {
	target  *url.URL
	random  bool
	status  store.Status
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
		p.status = store.STATUS_HEALTHY
	case "failure":
		p.status = store.STATUS_FAILURE
	case "unknown":
		p.status = store.STATUS_UNKNOWN
	case "random":
		p.random = true
	default:
		return DummyProbe{}, errors.New("opaque must healthy, failure, unknown, or random")
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

func (p DummyProbe) Status() store.Status {
	if !p.random {
		return p.status
	}

	return []store.Status{
		store.STATUS_HEALTHY,
		store.STATUS_UNKNOWN,
		store.STATUS_FAILURE,
	}[rand.Intn(3)]
}

func (p DummyProbe) Target() *url.URL {
	return p.target
}

func (p DummyProbe) Check(ctx context.Context) []store.Record {
	stime := time.Now()

	r := []store.Record{{
		CheckedAt: stime,
		Status:    p.Status(),
		Target:    p.target,
		Latency:   p.latency,
		Message:   p.message,
	}}

	if p.latency > 0 {
		select {
		case <-time.After(p.latency):
		case <-ctx.Done():
			r[0].Latency = time.Now().Sub(stime)
			r[0].Status = store.STATUS_UNKNOWN
			r[0].Message = "timed out or interrupted"
		}
	} else {
		r[0].Latency = time.Now().Sub(stime)
	}

	return r
}
