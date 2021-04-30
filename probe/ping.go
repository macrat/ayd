package probe

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/macrat/ayd/store"
	"github.com/macrat/go-parallel-pinger"
)

var (
	pingerV4 *pinger.Pinger = nil
	pingerV6 *pinger.Pinger = nil
)

func StartPinger(ctx context.Context) error {
	pingerV4 = pinger.NewIPv4()
	pingerV6 = pinger.NewIPv6()

	if err := pingerV4.Start(ctx); err != nil {
		return err
	}

	if err := pingerV6.Start(ctx); err != nil {
		return err
	}

	return nil
}

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

func (p PingProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	target, err := net.ResolveIPAddr("ip", p.target.Opaque)
	if err != nil {
		r.Report(store.Record{
			CheckedAt: time.Now(),
			Target:    p.target,
			Status:    store.STATUS_UNKNOWN,
			Message:   err.Error(),
		})
		return
	}

	ping := pingerV4
	if target.IP.To4() == nil {
		ping = pingerV6
	}

	startTime := time.Now()
	result, err := ping.Ping(ctx, target, 4, 500*time.Millisecond)

	status := store.STATUS_FAILURE
	if result.Loss == 0 {
		status = store.STATUS_HEALTHY
	}

	r.Report(timeoutOr(ctx, store.Record{
		CheckedAt: startTime,
		Target:    p.target,
		Status:    status,
		Message: fmt.Sprintf(
			"rtt(min/avg/max)=%.2f/%.2f/%.2f send/rcv=%d/%d",
			float64(result.MinRTT.Microseconds())/1000,
			float64(result.AvgRTT.Microseconds())/1000,
			float64(result.MaxRTT.Microseconds())/1000,
			result.Sent,
			result.Recv,
		),
		Latency: result.AvgRTT,
	}))
}
