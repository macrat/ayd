package scheme

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrTCPPortMissing = errors.New("TCP target's port number is required")
)

type TCPScheme struct {
	target *url.URL
}

func NewTCPScheme(u *url.URL) (TCPScheme, error) {
	scheme, separator, _ := SplitScheme(u.Scheme)

	if separator != 0 {
		return TCPScheme{}, ErrUnsupportedScheme
	}

	s := TCPScheme{&url.URL{Scheme: scheme, Host: u.Host, Fragment: u.Fragment}}
	if u.Host == "" {
		s.target.Host = u.Opaque
	}

	if s.target.Hostname() == "" {
		return TCPScheme{}, ErrMissingHost
	}
	if s.target.Port() == "" {
		return TCPScheme{}, ErrTCPPortMissing
	}

	return s, nil
}

func (s TCPScheme) Target() *url.URL {
	return s.target
}

func (s TCPScheme) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dialer net.Dialer

	st := time.Now()
	conn, err := dialer.DialContext(ctx, s.target.Scheme, s.target.Host)
	d := time.Now().Sub(st)

	rec := api.Record{
		CheckedAt: st,
		Target:    s.target,
		Latency:   d,
	}

	if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = err.Error()

		dnsErr := &net.DNSError{}

		if errors.Is(err, &net.AddrError{}) {
			rec.Status = api.StatusUnknown
		} else if errors.As(err, &dnsErr) {
			rec.Status = api.StatusUnknown
			rec.Message = dnsErrorToMessage(dnsErr)
		}
	} else {
		rec.Status = api.StatusHealthy
		rec.Message = "source=" + conn.LocalAddr().String() + " target=" + conn.RemoteAddr().String()
		conn.Close()
	}

	r.Report(s.target, timeoutOr(ctx, rec))
}

func (s TCPScheme) Alert(ctx context.Context, r Reporter, _ api.Record) {
	s.Probe(ctx, AlertReporter{s.target, r})
}
