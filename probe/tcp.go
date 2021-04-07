package probe

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

func TCPProbe(u *url.URL) Result {
	st := time.Now()
	conn, err := net.DialTimeout("tcp", u.Opaque, 10*time.Second)
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
		Target:    u,
		Status:    status,
		Message:   message,
		Latency:   d,
	}
}
