package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/go-parallel-pinger"
)

var (
	ErrFailedToPreparePing = errors.New("failed to setup ping service")
)

type ResourceLocker struct {
	sync.Mutex

	doneSignal *sync.Cond
	count      int
	teardown   func()
}

func NewResourceLocker() *ResourceLocker {
	rl := &ResourceLocker{}
	rl.doneSignal = sync.NewCond(rl)
	return rl
}

func (rl *ResourceLocker) Start(prepareResource func() (teardown func(), err error)) error {
	rl.Lock()
	defer rl.Unlock()

	if rl.count == 0 {
		var err error
		rl.teardown, err = prepareResource()
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

		if rl.count == 0 {
			rl.teardown()
		}
	}
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

func getAydPrivilegedEnv() bool {
	switch strings.ToLower(os.Getenv("AYD_PRIVILEGED")) {
	case "", "0", "no", "false":
		return false
	}
	return true
}

func makePingers() (v4, v6 *pinger.Pinger) {
	v4 = pinger.NewIPv4()
	v6 = pinger.NewIPv6()

	if getAydPrivilegedEnv() {
		v4.SetPrivileged(true)
		v6.SetPrivileged(true)
	}

	return v4, v6
}

func (p *autoPingerStruct) start() (teardown func(), err error) {
	p.v4, p.v6 = makePingers()

	ctx, stop := context.WithCancel(context.Background())

	if err := p.v4.Start(ctx); err != nil {
		stop()
		p.v4 = nil
		p.v6 = nil
		return nil, err
	}

	if err := p.v6.Start(ctx); err != nil {
		stop()
		p.v4 = nil
		p.v6 = nil
		return nil, err
	}

	return func() {
		stop()
		p.v4 = nil
		p.v6 = nil
	}, nil
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
	scheme, separator, _ := SplitScheme(u.Scheme)
	if separator != 0 {
		return PingProbe{}, ErrUnsupportedScheme
	}

	if err := CheckPingPermission(); err != nil {
		return PingProbe{}, ayderr.New(ErrFailedToPreparePing, err, ErrFailedToPreparePing.Error())
	}

	if u.Opaque != "" {
		return PingProbe{&url.URL{Scheme: scheme, Opaque: u.Opaque, Fragment: u.Fragment}}, nil
	} else if u.Hostname() != "" {
		return PingProbe{&url.URL{Scheme: scheme, Opaque: u.Hostname(), Fragment: u.Fragment}}, nil
	} else {
		return PingProbe{}, ErrMissingHost
	}
}

func (p PingProbe) Target() *url.URL {
	return p.target
}

func (p PingProbe) Check(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	proto := "ip"
	if p.target.Scheme == "ping4" {
		proto = "ip4"
	} else if p.target.Scheme == "ping6" {
		proto = "ip6"
	}

	target, err := net.ResolveIPAddr(proto, p.target.Opaque)
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
		Message: fmt.Sprintf(
			"ip=%s rtt(min/avg/max)=%.2f/%.2f/%.2f send/recv=%d/%d",
			target,
			float64(result.MinRTT.Microseconds())/1000,
			float64(result.AvgRTT.Microseconds())/1000,
			float64(result.MaxRTT.Microseconds())/1000,
			result.Sent,
			result.Recv,
		),
		Latency: result.AvgRTT,
	}

	switch result.Loss {
	case 0:
		rec.Status = api.StatusHealthy
	case 3:
		rec.Status = api.StatusFailure
	default:
		rec.Status = api.StatusDebased
	}

	if ctx.Err() == context.Canceled {
		rec.Status = api.StatusAborted
		rec.Message = "probe aborted"
		rec.Latency = d
	}

	r.Report(rec)
}
