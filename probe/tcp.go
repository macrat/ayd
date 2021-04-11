package probe

import (
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

func NewTCPProbe(u *url.URL) TCPProbe {
	if u.Opaque != "" {
		return TCPProbe{&url.URL{Scheme: "tcp", Opaque: u.Opaque}}
	} else {
		return TCPProbe{&url.URL{Scheme: "tcp", Opaque: u.Host}}
	}
}

func (p TCPProbe) Target() *url.URL {
	return p.target
}

func (p TCPProbe) Check() store.Record {
	st := time.Now()
	conn, err := net.DialTimeout("tcp", p.target.Opaque, 10*time.Second)
	d := time.Now().Sub(st)

	r := store.Record{
		CheckedAt: st,
		Target:    p.target,
		Latency:   d,
	}

	if err != nil {
		r.Status = store.STATUS_FAIL
		r.Message = err.Error()
		if _, ok := errors.Unwrap(err).(*net.AddrError); ok {
			r.Status = store.STATUS_UNKNOWN
		}
		if e, ok := errors.Unwrap(err).(*net.DNSError); ok && e.IsNotFound {
			r.Status = store.STATUS_UNKNOWN
		}
	} else {
		r.Status = store.STATUS_OK
		r.Message = fmt.Sprintf("%s -> %s", conn.LocalAddr(), conn.RemoteAddr())
		conn.Close()
	}

	return r
}
