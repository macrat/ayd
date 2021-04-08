package probe

import (
	"net"
	"net/url"
	"strings"
	"time"
)

type DNSProbe struct {
	target *url.URL
}

func NewDNSProbe(u *url.URL) DNSProbe {
	if u.Opaque != "" {
		return DNSProbe{&url.URL{Scheme: "dns", Opaque: u.Opaque}}
	} else {
		return DNSProbe{&url.URL{Scheme: "dns", Opaque: u.Hostname()}}
	}
}

func (p DNSProbe) Target() *url.URL {
	return p.target
}

func (p DNSProbe) Check() Result {
	st := time.Now()
	addrs, err := net.LookupHost(p.target.Opaque)
	d := time.Now().Sub(st)

	var status Status
	var message string
	if err != nil {
		status = STATUS_FAIL
		message = err.Error()
	} else {
		status = STATUS_OK
		message = strings.Join(addrs, ", ")
	}

	return Result{
		CheckedAt: st,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
