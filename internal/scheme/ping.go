package scheme

import (
	"context"
	"errors"
	"net"
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

func pingSettings() (count int, interval, timeout time.Duration, privileged *bool) {
	var err error

	count, err = strconv.Atoi(os.Getenv("AYD_PING_PACKETS"))
	if err != nil || count <= 0 {
		count = 3
	} else if count >= 100 {
		count = 100
	}

	d, err := time.ParseDuration(os.Getenv("AYD_PING_PERIOD"))
	if err != nil || d <= 0 {
		d = time.Second
	} else if d > 30*time.Minute {
		d = 30 * time.Minute
	}
	interval = d / time.Duration(count)

	timeout = d + 30*time.Second

	pri := strings.ToLower(os.Getenv("AYD_PING_PRIVILEGED"))
	if pri == "1" || pri == "true" || pri == "yes" || pri == "on" {
		p := true
		privileged = &p
	} else if pri == "0" || pri == "false" || pri == "no" || pri == "off" {
		p := false
		privileged = &p
	}

	return
}

type startStoper interface {
	Start() error
	Stop()
}

type sharedResource[T startStoper] struct {
	sync.Mutex

	doneSignal *sync.Cond
	count      int

	resource T
}

func newSharedResource[T startStoper](resource T) *sharedResource[T] {
	sr := &sharedResource[T]{
		resource: resource,
	}
	sr.doneSignal = sync.NewCond(sr)
	return sr
}

func (sr *sharedResource[T]) Get() (resource T, err error) {
	sr.Lock()
	defer sr.Unlock()

	if sr.count == 0 {
		err = sr.resource.Start()
		if err != nil {
			return
		}
	}

	sr.count++

	return sr.resource, nil
}

func (sr *sharedResource[T]) Release() {
	sr.Lock()
	defer sr.Unlock()

	if sr.count > 0 {
		sr.count--

		if sr.count == 0 {
			sr.resource.Stop()
		}
	}
}

type simplePinger struct {
	v4   *pinger.Pinger
	v6   *pinger.Pinger
	stop context.CancelFunc
}

func (p *simplePinger) Start() error {
	_, _, _, privileged := pingSettings()

	p.v4 = pinger.NewIPv4()
	p.v6 = pinger.NewIPv6()

	if privileged != nil {
		p.v4.SetPrivileged(*privileged)
		p.v6.SetPrivileged(*privileged)
	}

	ctx, stop := context.WithCancel(context.Background())
	p.stop = stop

	err := p.startPingers(ctx)
	if err != nil {
		p.Stop()
		return err
	}

	return nil
}

func (p *simplePinger) Stop() {
	p.v4 = nil
	p.v6 = nil
	p.stop()
	p.stop = nil
}

type autoPingerStruct struct {
	sr *sharedResource[*simplePinger]
}

func newAutoPinger() *autoPingerStruct {
	return &autoPingerStruct{
		sr: newSharedResource(&simplePinger{}),
	}
}

func (p *autoPingerStruct) getFor(target net.IP) (*pinger.Pinger, error) {
	pinger, err := p.sr.Get()
	if err != nil {
		return nil, err
	}

	if target.To4() != nil {
		return pinger.v4, nil
	}
	return pinger.v6, nil
}

func (p *autoPingerStruct) Ping(ctx context.Context, target *net.IPAddr) (startTime time.Time, duration time.Duration, result pinger.Result, err error) {
	defer p.sr.Release()

	ping, err := p.getFor(target.IP)
	if err != nil {
		return time.Now(), 0, pinger.Result{}, err
	}

	packets, interval, _, _ := pingSettings()

	startTime = time.Now()
	result, err = ping.Ping(ctx, target, packets, interval)
	duration = time.Since(startTime)

	return
}

func (p *autoPingerStruct) Test() error {
	_, err := p.sr.Get()
	if err != nil {
		return err
	}
	p.sr.Release()
	return nil
}

func pingResultToRecord(ctx context.Context, target *api.URL, startTime time.Time, result pinger.Result) api.Record {
	rec := api.Record{
		Time:    startTime,
		Latency: result.AvgRTT,
		Target:  target,
		Extra: map[string]interface{}{
			"rtt_min":      float64(result.MinRTT.Microseconds()) / 1000,
			"rtt_avg":      float64(result.AvgRTT.Microseconds()) / 1000,
			"rtt_max":      float64(result.MaxRTT.Microseconds()) / 1000,
			"packets_recv": result.Recv,
			"packets_sent": result.Sent,
		},
	}

	switch {
	case result.Loss == 0:
		rec.Status = api.StatusHealthy
		rec.Message = "all packets came back"
	case result.Recv == 0:
		rec.Status = api.StatusFailure
		rec.Message = "all packets have dropped"
	default:
		rec.Status = api.StatusDegrade
		rec.Message = "some packets have dropped"
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

// PingProbe is a Prober implementation for SNMP echo request aka ping.
type PingProbe struct {
	target *api.URL
}

func NewPingProbe(u *api.URL) (PingProbe, error) {
	scheme, separator, _ := SplitScheme(u.Scheme)
	if separator != 0 {
		return PingProbe{}, ErrUnsupportedScheme
	}

	if err := autoPinger.Test(); err != nil {
		return PingProbe{}, ayderr.New(ErrFailedToPreparePing, err, ErrFailedToPreparePing.Error())
	}

	if u.Opaque != "" {
		return PingProbe{&api.URL{Scheme: scheme, Opaque: strings.ToLower(u.Opaque), Fragment: u.Fragment}}, nil
	} else if u.ToURL().Hostname() != "" {
		return PingProbe{&api.URL{Scheme: scheme, Opaque: strings.ToLower(u.ToURL().Hostname()), Fragment: u.Fragment}}, nil
	} else {
		return PingProbe{}, ErrMissingHost
	}
}

func (s PingProbe) Target() *api.URL {
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
	_, _, timeout, _ := pingSettings()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	preparingError := func(err error) {
		r.Report(s.target, api.Record{
			Time:    time.Now(),
			Target:  s.target,
			Status:  api.StatusUnknown,
			Message: err.Error(),
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
