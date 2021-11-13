package scheme

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrTCPPortMissing = errors.New("TCP target's port number is required")
)

// TCPProbe is a Prober implementation for the TCP.
type TCPProbe struct {
	target *url.URL
}

func NewTCPProbe(u *url.URL) (TCPProbe, error) {
	scheme, separator, _ := SplitScheme(u.Scheme)

	if separator != 0 {
		return TCPProbe{}, ErrUnsupportedScheme
	}

	s := TCPProbe{&url.URL{Scheme: scheme, Host: strings.ToLower(u.Host), Fragment: u.Fragment}}
	if u.Host == "" {
		s.target.Host = strings.ToLower(u.Opaque)
	}

	if s.target.Hostname() == "" {
		return TCPProbe{}, ErrMissingHost
	}
	if s.target.Port() == "" {
		return TCPProbe{}, ErrTCPPortMissing
	}

	return s, nil
}

func (s TCPProbe) Target() *url.URL {
	return s.target
}

func (s TCPProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dialer net.Dialer

	st := time.Now()
	conn, err := dialer.DialContext(ctx, s.target.Scheme, s.target.Host)
	d := time.Since(st)

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
