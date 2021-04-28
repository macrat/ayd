package probe

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/macrat/ayd/store"
)

type TCPProbe struct {
	target *url.URL
}

func NewTCPProbe(u *url.URL) (TCPProbe, error) {
	p := TCPProbe{&url.URL{Scheme: "tcp", Opaque: u.Opaque}}
	if u.Opaque == "" {
		p.target.Opaque = u.Host
	}
	if _, _, err := net.SplitHostPort(p.target.Opaque); err != nil {
		return TCPProbe{}, err
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
	conn, err := dialer.DialContext(ctx, "tcp", p.target.Opaque)
	d := time.Now().Sub(st)

	rec := store.Record{
		CheckedAt: st,
		Target:    p.target,
		Latency:   d,
	}

	if err != nil {
		rec.Status = store.STATUS_FAILURE
		rec.Message = err.Error()
		if _, ok := errors.Unwrap(err).(*net.AddrError); ok {
			rec.Status = store.STATUS_UNKNOWN
		}
		if e, ok := errors.Unwrap(err).(*net.DNSError); ok && e.IsNotFound {
			rec.Status = store.STATUS_UNKNOWN
		}
		if e := errors.Unwrap(err); e != nil && e.Error() == "operation was canceled" {
			rec.Status = store.STATUS_UNKNOWN
			rec.Message = "timed out or interrupted"
		}
	} else {
		rec.Status = store.STATUS_HEALTHY
		rec.Message = conn.LocalAddr().String() + " -> " + conn.RemoteAddr().String()
		conn.Close()
	}

	r.Report(rec)
}
