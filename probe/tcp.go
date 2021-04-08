package probe

import (
	"fmt"
	"net"
	"net/url"
	"time"
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

func (p TCPProbe) Check() Result {
	st := time.Now()
	conn, err := net.DialTimeout("tcp", p.target.Opaque, 10*time.Second)
	d := time.Now().Sub(st)

	var status Status
	var message string
	if err != nil {
		status = STATUS_FAIL
		message = err.Error()
	} else {
		status = STATUS_OK
		message = fmt.Sprintf("%s -> %s", conn.LocalAddr(), conn.RemoteAddr())
		conn.Close()
	}

	return Result{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
