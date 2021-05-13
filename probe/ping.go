package probe

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/go-parallel-pinger"
)

type pingerManagerStruct struct {
	sync.Mutex

	v4   *pinger.Pinger
	v6   *pinger.Pinger
	stop func()
}

func (p *pingerManagerStruct) Start(ctx context.Context) error {
	p.v4 = pinger.NewIPv4()
	p.v6 = pinger.NewIPv6()

	if os.Getenv("AYD_PRIVILEGED") != "" {
		p.v4.SetPrivileged(true)
		p.v6.SetPrivileged(true)
	}

	ctx, stop := context.WithCancel(ctx)
	p.stop = stop

	if err := p.v4.Start(ctx); err != nil {
		p.v4 = nil
		p.v6 = nil
		stop()
		return err
	}

	if err := p.v6.Start(ctx); err != nil {
		p.v4 = nil
		p.v6 = nil
		stop()
		return err
	}

	return nil
}

func (p *pingerManagerStruct) Stop() {
	p.stop()
}

func (p *pingerManagerStruct) GetFor(target net.IP) (*pinger.Pinger, error) {
	p.Lock()
	defer p.Unlock()

	if p.v4 == nil {
		err := p.Start(context.Background()) // XXX: there is no way to stop pinger
		if err != nil {
			return nil, err
		}
	}

	if target.To4() != nil {
		return p.v4, nil
	}
	return p.v6, nil
}

var (
	pingerManager = &pingerManagerStruct{}
)

func StartPinger(ctx context.Context) error {
	return pingerManager.Start(ctx)
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
		r.Report(api.Record{
			CheckedAt: time.Now(),
			Target:    p.target,
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
		return
	}

	ping, err := pingerManager.GetFor(target.IP)
	if err != nil {
		r.Report(api.Record{
			CheckedAt: time.Now(),
			Target:    p.target,
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
		return
	}

	startTime := time.Now()
	result, err := ping.Ping(ctx, target, 4, 500*time.Millisecond)
	d := time.Now().Sub(startTime)

	rec := api.Record{
		CheckedAt: startTime,
		Target:    p.target,
		Status:    api.StatusFailure,
		Message: fmt.Sprintf(
			"rtt(min/avg/max)=%.2f/%.2f/%.2f send/recv=%d/%d",
			float64(result.MinRTT.Microseconds())/1000,
			float64(result.AvgRTT.Microseconds())/1000,
			float64(result.MaxRTT.Microseconds())/1000,
			result.Sent,
			result.Recv,
		),
		Latency: result.AvgRTT,
	}

	if result.Loss == 0 {
		rec.Status = api.StatusHealthy
	}

	if ctx.Err() == context.Canceled {
		rec.Status = api.StatusAborted
		rec.Message = "probe aborted"
		rec.Latency = d
	}

	r.Report(rec)
}
