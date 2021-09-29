package probe

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

type TCPProbe struct {
	target *url.URL
}

func NewTCPProbe(u *url.URL) (TCPProbe, error) {
	scheme := strings.SplitN(u.Scheme, "-", 2)[0]
	scheme = strings.SplitN(scheme, "+", 2)[0]

	p := TCPProbe{&url.URL{Scheme: scheme, Host: u.Host, Fragment: u.Fragment}}
	if u.Host == "" {
		p.target.Host = u.Opaque
	}
	if port := p.target.Port(); port == "" {
		return TCPProbe{}, ErrTCPPortMissing
	}
	return p, nil
}

func (p TCPProbe) Target() *url.URL {
	return p.target
}

func (p TCPProbe) Check(ctx context.Context, r Reporter) {
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
		if _, ok := errors.Unwrap(err).(*net.AddrError); ok {
			rec.Status = api.StatusUnknown
		}
		if e, ok := errors.Unwrap(err).(*net.DNSError); ok && e.IsNotFound {
			rec.Status = api.StatusUnknown
		}
	} else {
		rec.Status = api.StatusHealthy
		rec.Message = "source=" + conn.LocalAddr().String() + " target=" + conn.RemoteAddr().String()
		conn.Close()
	}

	r.Report(timeoutOr(ctx, rec))
}
