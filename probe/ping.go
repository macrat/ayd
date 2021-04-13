package probe

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-ping/ping"
	"github.com/macrat/ayd/store"
)

type PingProbe struct {
	target *url.URL
}

func NewPingProbe(u *url.URL) (PingProbe, error) {
	if u.Opaque != "" {
		return PingProbe{&url.URL{Scheme: "ping", Opaque: u.Opaque}}, nil
	} else {
		return PingProbe{&url.URL{Scheme: "ping", Opaque: u.Hostname()}}, nil
	}
}

func (p PingProbe) Target() *url.URL {
	return p.target
}

func (p PingProbe) Check() []store.Record {
	pinger, err := ping.NewPinger(p.target.Opaque)
	if err != nil {
		status := store.STATUS_FAILURE

		if e, ok := err.(*net.DNSError); ok && e.IsNotFound {
			status = store.STATUS_UNKNOWN
		}

		return []store.Record{{
			CheckedAt: time.Now(),
			Target:    p.target,
			Status:    status,
			Message:   err.Error(),
		}}
	}

	pinger.Interval = 500 * time.Millisecond
	pinger.Timeout = 10 * time.Second
	pinger.Count = 4
	pinger.Debug = true

	startTime := time.Now()

	pinger.Run()

	stat := pinger.Statistics()

	status := store.STATUS_FAILURE
	if stat.PacketLoss == 0 {
		status = store.STATUS_HEALTHY
	}

	return []store.Record{{
		CheckedAt: startTime,
		Target:    p.target,
		Status:    status,
		Message: fmt.Sprintf(
			"rtt(min/avg/max)=%.2f/%.2f/%.2f send/rcv=%d/%d",
			float64(stat.MinRtt.Microseconds())/1000,
			float64(stat.AvgRtt.Microseconds())/1000,
			float64(stat.MaxRtt.Microseconds())/1000,
			pinger.PacketsSent,
			pinger.PacketsRecv,
		),
		Latency: stat.AvgRtt,
	}}
}
