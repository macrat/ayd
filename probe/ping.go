package probe

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-ping/ping"
)

func PingProbe(u *url.URL) Result {
	pinger, err := ping.NewPinger(u.Opaque)
	if err != nil {
		return Result{
			CheckedAt: time.Now(),
			Target:    u,
			Status:    STATUS_FAIL,
			Message:   err.Error(),
		}
	}

	pinger.Interval = 500 * time.Millisecond
	pinger.Timeout = 10 * time.Second
	pinger.Count = 4
	pinger.Debug = true

	startTime := time.Now()

	err = pinger.Run()
	if err != nil {
		fmt.Println(err)
	}

	stat := pinger.Statistics()

	status := STATUS_FAIL
	if stat.PacketLoss == 0 {
		status = STATUS_OK
	}

	return Result{
		CheckedAt: startTime,
		Target:    u,
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
	}
}
