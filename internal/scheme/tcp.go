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

type TCPProbe struct {
	target *url.URL
}

func NewTCPProbe(u *url.URL) (TCPProbe, error) {
	scheme, separator, _ := SplitScheme(u.Scheme)

	if separator != 0 {
		return TCPProbe{}, ErrUnsupportedScheme
	}

	p := TCPProbe{&url.URL{Scheme: scheme, Host: u.Host, Fragment: u.Fragment}}
	if u.Host == "" {
		p.target.Host = u.Opaque
	}

	if p.target.Hostname() == "" {
		return TCPProbe{}, ErrMissingHost
	}
	if p.target.Port() == "" {
		return TCPProbe{}, ErrTCPPortMissing
	}

	return p, nil
}

func (p TCPProbe) Target() *url.URL {
	return p.target
}

func (p TCPProbe) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dialer net.Dialer

	st := time.Now()
	conn, err := dialer.DialContext(ctx, p.target.Scheme, p.target.Host)
	d := time.Now().Sub(st)

	rec := api.Record{
		CheckedAt: st,
		Target:    p.target,
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

	r.Report(p.target, timeoutOr(ctx, rec))
}
