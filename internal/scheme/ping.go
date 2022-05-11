package scheme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
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

func pingSettings() (count int, interval, timeout time.Duration) {
	var err error

	count, err = strconv.Atoi(os.Getenv("AYD_PING_PACKETS"))
	if err != nil || count <= 0 {
		count = 3
	} else if count == 50 {
		count = 100
	}

	d, err := time.ParseDuration(os.Getenv("AYD_PING_PERIOD"))
	if err != nil || d <= 0 {
		d = time.Second
	} else if d > 30*time.Minute {
		d = 30 * time.Minute
	}
	interval = d / time.Duration(count)

	timeout = d * 2
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}

	return
}

type resourceLocker struct {
	sync.Mutex

	doneSignal *sync.Cond
	count      int
	teardown   func()
}

func newResourceLocker() *resourceLocker {
	rl := &resourceLocker{}
	rl.doneSignal = sync.NewCond(rl)
	return rl
}

func (rl *resourceLocker) Start(prepareResource func() (teardown func(), err error)) error {
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

func (rl *resourceLocker) Done() {
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
	rl *resourceLocker
	v4 *pinger.Pinger
	v6 *pinger.Pinger
}

func newAutoPinger() *autoPingerStruct {
	return &autoPingerStruct{
		rl: newResourceLocker(),
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

	packets, interval, _ := pingSettings()

	startTime = time.Now()
	result, err = ping.Ping(ctx, target, packets, interval)
	duration = time.Since(startTime)

	return
}

func pingResultToRecord(ctx context.Context, target *url.URL, startTime time.Time, result pinger.Result) api.Record {
	rec := api.Record{
		CheckedAt: startTime,
		Latency:   result.AvgRTT,
		Target:    target,
		Message: fmt.Sprintf(
			"ip=%s rtt(min/avg/max)=%.2f/%.2f/%.2f recv/sent=%d/%d",
			result.Target,
			float64(result.MinRTT.Microseconds())/1000,
			float64(result.AvgRTT.Microseconds())/1000,
			float64(result.MaxRTT.Microseconds())/1000,
			result.Recv,
			result.Sent,
		),
	}

	switch {
	case result.Loss == 0:
		rec.Status = api.StatusHealthy
	case result.Recv == 0:
		rec.Status = api.StatusFailure
	default:
		rec.Status = api.StatusDegrade
	}

	if ctx.Err() == context.Canceled {
		rec.Status = api.StatusAborted
		rec.Message = "probe aborted"
	}

	return rec
}

var (
	autoPinger = newAutoPinger()
)

// checkPingPermission tries to prepare pinger for check if it has permission.
func checkPingPermission() error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	p, _ := makePingers()
	return p.Start(ctx)
}

// PingProbe is a Prober implementation for SNMP echo request aka ping.
type PingProbe struct {
	target *url.URL
}

func NewPingProbe(u *url.URL) (PingProbe, error) {
	scheme, separator, _ := SplitScheme(u.Scheme)
	if separator != 0 {
		return PingProbe{}, ErrUnsupportedScheme
	}

	if err := checkPingPermission(); err != nil {
		return PingProbe{}, ayderr.New(ErrFailedToPreparePing, err, ErrFailedToPreparePing.Error())
	}

	if u.Opaque != "" {
		return PingProbe{&url.URL{Scheme: scheme, Opaque: strings.ToLower(u.Opaque), Fragment: u.Fragment}}, nil
	} else if u.Hostname() != "" {
		return PingProbe{&url.URL{Scheme: scheme, Opaque: strings.ToLower(u.Hostname()), Fragment: u.Fragment}}, nil
	} else {
		return PingProbe{}, ErrMissingHost
	}
}

func (s PingProbe) Target() *url.URL {
	return s.target
}

func (s PingProbe) proto() string {
	switch s.target.Scheme {
	case "ping4":
		return "ip4"
	case "ping6":
		return "ip6"
	default:
		return "ip"
	}
}

func (s PingProbe) Probe(ctx context.Context, r Reporter) {
	_, _, timeout := pingSettings()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	preparingError := func(err error) {
		r.Report(s.target, api.Record{
			CheckedAt: time.Now(),
			Target:    s.target,
			Status:    api.StatusUnknown,
			Message:   err.Error(),
		})
	}

	target, err := net.ResolveIPAddr(s.proto(), s.target.Opaque)
	if err != nil {
		preparingError(err)
		return
	}

	stime, d, result, err := autoPinger.Ping(ctx, target)
	if err != nil {
		preparingError(err)
		return
	}

	rec := pingResultToRecord(ctx, s.target, stime, result)
	if rec.Status == api.StatusAborted {
		rec.Latency = d
	}

	r.Report(s.target, rec)
}
