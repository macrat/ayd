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

type ResourceLocker struct {
	sync.Mutex

	doneSignal *sync.Cond
	count      int
}

func NewResourceLocker() *ResourceLocker {
	rl := &ResourceLocker{}
	rl.doneSignal = sync.NewCond(rl)
	return rl
}

func (rl *ResourceLocker) Start(prepareResource func() error) error {
	rl.Lock()
	defer rl.Unlock()

	if rl.count == 0 {
		err := prepareResource()
		if err != nil {
			return err
		}
	}

	rl.count++

	return nil
}

func (rl *ResourceLocker) Done() {
	rl.Lock()
	defer rl.Unlock()

	if rl.count > 0 {
		rl.count--
	}

	rl.doneSignal.Broadcast()
}

func (rl *ResourceLocker) Teardown(f func()) {
	rl.Lock()
	defer rl.Unlock()

	for rl.count > 0 {
		rl.doneSignal.Wait()
	}

	f()
}

type autoPingerStruct struct {
	rl *ResourceLocker
	v4 *pinger.Pinger
	v6 *pinger.Pinger
}

func newAutoPinger() *autoPingerStruct {
	return &autoPingerStruct{
		rl: NewResourceLocker(),
	}
}

func makePingers() (v4, v6 *pinger.Pinger) {
	v4 = pinger.NewIPv4()
	v6 = pinger.NewIPv6()

	if os.Getenv("AYD_PRIVILEGED") != "" {
		v4.SetPrivileged(true)
		v6.SetPrivileged(true)
	}

	return v4, v6
}

func (p *autoPingerStruct) start() error {
	p.v4, p.v6 = makePingers()

	ctx, stop := context.WithCancel(context.Background())

	if err := p.v4.Start(ctx); err != nil {
		stop()
		p.v4 = nil
		p.v6 = nil
		return err
	}

	if err := p.v6.Start(ctx); err != nil {
		stop()
		p.v4 = nil
		p.v6 = nil
		return err
	}

	go p.rl.Teardown(func() {
		stop()
		p.v4 = nil
		p.v6 = nil
	})

	return nil
}

func (p *autoPingerStruct) getFor(target net.IP) (*pinger.Pinger, error) {
	if err := p.rl.Start(p.start); err != nil {
		return nil, err
	}

	if target.To4() != nil {
		return p.v4, nil
	}
	return p.v6, nil
}

func (p *autoPingerStruct) Ping(ctx context.Context, target *net.IPAddr) (startTime time.Time, duration time.Duration, result pinger.Result, err error) {
	defer p.rl.Done()

	ping, err := p.getFor(target.IP)
	if err != nil {
		return time.Now(), 0, pinger.Result{}, err
	}

	startTime = time.Now()
	result, err = ping.Ping(ctx, target, 3, 500*time.Millisecond)
	duration = time.Now().Sub(startTime)

	return
}

var (
	autoPinger = newAutoPinger()
)

func CheckPingPermission() error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	p, _ := makePingers()
	return p.Start(ctx)
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

	stime, d, result, err := autoPinger.Ping(ctx, target)
	if err != nil {
		r.Report(api.Record{
			CheckedAt: time.Now(),
			Target:    p.target,
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
		return
	}

	rec := api.Record{
		CheckedAt: stime,
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
