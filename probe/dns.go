package probe

import (
	"net"
	"net/url"
	"strings"
	"time"
)

func DNSProbe(u *url.URL) Result {
	st := time.Now()
	addrs, err := net.LookupHost(u.Opaque)
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
		Target:    u,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
