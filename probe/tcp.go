package probe

import (
	"context"
	"errors"
	"fmt"
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

func (p TCPProbe) Check(ctx context.Context) []store.Record {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var dialer net.Dialer

	st := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", p.target.Opaque)
	d := time.Now().Sub(st)

	r := store.Record{
		CheckedAt: st,
		Target:    p.target,
		Latency:   d,
	}

	if err != nil {
		r.Status = store.STATUS_FAILURE
		r.Message = err.Error()
		if _, ok := errors.Unwrap(err).(*net.AddrError); ok {
			r.Status = store.STATUS_UNKNOWN
		}
		if e, ok := errors.Unwrap(err).(*net.DNSError); ok && e.IsNotFound {
			r.Status = store.STATUS_UNKNOWN
		}
	} else {
		r.Status = store.STATUS_HEALTHY
		r.Message = fmt.Sprintf("%s -> %s", conn.LocalAddr(), conn.RemoteAddr())
		conn.Close()
	}

	return []store.Record{r}
}
