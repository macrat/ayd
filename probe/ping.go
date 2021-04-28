package probe

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

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

func (p PingProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pinger, err := getPinger(p.target.Opaque)
	if err != nil {
		status := store.STATUS_FAILURE

		if e, ok := err.(*net.DNSError); ok && e.IsNotFound {
			status = store.STATUS_UNKNOWN
		}

		r.Report(store.Record{
			CheckedAt: time.Now(),
			Target:    p.target,
			Status:    status,
			Message:   err.Error(),
		})
		return
	}

	pinger.Interval = 500 * time.Millisecond
	pinger.Timeout = 10 * time.Second
	pinger.Count = 4
	pinger.Debug = true

	go func() {
		<-ctx.Done()
		pinger.Stop()
	}()

	startTime := time.Now()

	err = pinger.Run()
	if err != nil {
		r.Report(store.Record{
			CheckedAt: startTime,
			Target:    p.target,
			Status:    store.STATUS_UNKNOWN,
			Message:   err.Error(),
			Latency:   time.Now().Sub(time.Now()),
		})
		return
	}

	stat := pinger.Statistics()

	status := store.STATUS_FAILURE
	if stat.PacketLoss == 0 {
		status = store.STATUS_HEALTHY
	}

	var message string
	select {
	case <-ctx.Done():
		status = store.STATUS_UNKNOWN
		message = "timed out or interrupted"
	default:
		message = fmt.Sprintf(
			"rtt(min/avg/max)=%.2f/%.2f/%.2f send/rcv=%d/%d",
			float64(stat.MinRtt.Microseconds())/1000,
			float64(stat.AvgRtt.Microseconds())/1000,
			float64(stat.MaxRtt.Microseconds())/1000,
			pinger.PacketsSent,
			pinger.PacketsRecv,
		)
	}

	r.Report(store.Record{
		CheckedAt: startTime,
		Target:    p.target,
		Status:    status,
		Message:   message,
		Latency:   stat.AvgRtt,
	})
}
