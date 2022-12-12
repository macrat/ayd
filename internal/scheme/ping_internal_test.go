package scheme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/go-parallel-pinger"
)

func TestPingSettings(t *testing.T) {
	tests := []struct {
		Env      [][]string
		Count    int
		Interval time.Duration
		Timeout  time.Duration
	}{
		{
			nil,
			3,
			time.Second / 3,
			31 * time.Second,
		},
		{
			[][]string{{"AYD_PING_PACKETS", "5"}},
			5,
			time.Second / 5,
			31 * time.Second,
		},
		{
			[][]string{{"AYD_PING_PACKETS", "-2"}},
			3,
			time.Second / 3,
			31 * time.Second,
		},
		{
			[][]string{{"AYD_PING_PACKETS", "123"}},
			100,
			time.Second / 100,
			31 * time.Second,
		},
		{
			[][]string{{"AYD_PING_PERIOD", "10m"}},
			3,
			10 * time.Minute / 3,
			630 * time.Second,
		},
		{
			[][]string{{"AYD_PING_PERIOD", "-10s"}},
			3,
			time.Second / 3,
			31 * time.Second,
		},
		{
			[][]string{{"AYD_PING_PERIOD", "3h"}},
			3,
			30 * time.Minute / 3,
			30*time.Minute + 30*time.Second,
		},
		{
			[][]string{
				{"AYD_PING_PACKETS", "42"},
				{"AYD_PING_PERIOD", "8m"},
			},
			42,
			8 * time.Minute / 42,
			8*time.Minute + 30*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.Env), func(t *testing.T) {
			for _, kv := range tt.Env {
				t.Setenv(kv[0], kv[1])
			}

			count, interval, timeout := pingSettings()

			if count != tt.Count {
				t.Errorf("expected %d packets but got %d", tt.Count, count)
			}

			if interval != tt.Interval {
				t.Errorf("expected %s interval but got %s", tt.Interval, interval)
			}

			if timeout != tt.Timeout {
				t.Errorf("expected %s timeout but got %s", tt.Timeout, timeout)
			}
		})
	}
}

func TestResourceLocker(t *testing.T) {
	rl := newResourceLocker()

	startCount := 0
	stopCount := 0

	start := func() {
		rl.Start(func() (func(), error) {
			startCount++
			return func() {
				stopCount++
			}, nil
		})
	}
	stop := func() {
		rl.Done()
	}

	assertCount := func(t *testing.T, start, stop int) {
		t.Helper()

		rl.Lock()
		defer rl.Unlock()

		if startCount != start || stopCount != stop {
			t.Errorf("unexpected count: start:%d stop:%d != start:%d stop:%d", startCount, stopCount, start, stop)
		}
	}

	start()
	start()
	assertCount(t, 1, 0)

	stop()
	assertCount(t, 1, 0)

	stop()
	assertCount(t, 1, 1)

	start()
	assertCount(t, 2, 1)

	stop()
	assertCount(t, 2, 2)

	stop()
	assertCount(t, 2, 2)
}

func TestResourceLocker_failedToStart(t *testing.T) {
	rl := newResourceLocker()

	want := errors.New("test error")

	err := rl.Start(func() (func(), error) {
		return nil, want
	})
	if err != want {
		t.Errorf("error wanted but got %s", err)
	}

	err = rl.Start(func() (func(), error) {
		return func() {}, nil
	})
	if err != nil {
		t.Errorf("error not wanted but got %s", err)
	}
}

func TestResourceLocker_flooding(t *testing.T) {
	rl := newResourceLocker()

	startCount := 0
	stopCount := 0

	start := func() {
		rl.Start(func() (func(), error) {
			startCount++
			return func() {
				stopCount++
			}, nil
		})
	}

	for i := 0; i < 10000; i++ {
		start()
	}
	for i := 0; i < 10000; i++ {
		rl.Done()
	}

	rl.Lock()
	defer rl.Unlock()

	if startCount != 1 {
		t.Errorf("unexpected start count: %d", startCount)
	}

	if stopCount != 1 {
		t.Errorf("unexpected stop count: %d", stopCount)
	}
}

func BenchmarkResourceLocker_Start(b *testing.B) {
	rl := newResourceLocker()

	starter := func() (func(), error) {
		return func() {}, nil
	}

	for i := 0; i < b.N; i++ {
		rl.Start(starter)
	}
}

func BenchmarkResourceLocker_Done(b *testing.B) {
	rl := newResourceLocker()

	starter := func() (func(), error) {
		return func() {}, nil
	}

	for i := 0; i < b.N; i++ {
		rl.Start(starter)
		rl.Done()
	}
}

func TestPingResultToRecord(t *testing.T) {
	t.Parallel()

	aliveCtx := context.Background()
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		Context   context.Context
		Target    *api.URL
		StartTime time.Time
		Result    pinger.Result
		Message   string
		Extra     map[string]interface{}
		Status    api.Status
	}{
		{
			aliveCtx,
			&api.URL{Scheme: "dummy-ping", Opaque: "healthy"},
			time.Now(),
			pinger.Result{
				Target: &net.IPAddr{net.IPv4(127, 0, 0, 1), ""},
				Sent:   3,
				Recv:   3,
				Loss:   0,
				MinRTT: 1234 * time.Microsecond,
				AvgRTT: 2345 * time.Microsecond,
				MaxRTT: 3456 * time.Microsecond,
			},
			"all packets came back",
			map[string]interface{}{
				"rtt_min":      1.234,
				"rtt_avg":      2.345,
				"rtt_max":      3.456,
				"packets_recv": 3,
				"packets_sent": 3,
			},
			api.StatusHealthy,
		},
		{
			aliveCtx,
			&api.URL{Scheme: "dummy-ping", Opaque: "failure"},
			time.Now().Add(-10 * time.Second),
			pinger.Result{
				Target: &net.IPAddr{net.IPv4(127, 1, 2, 3), ""},
				Sent:   3,
				Recv:   0,
				Loss:   3,
			},
			"all packets have dropped",
			map[string]interface{}{
				"rtt_min":      0.0,
				"rtt_avg":      0.0,
				"rtt_max":      0.0,
				"packets_recv": 0,
				"packets_sent": 3,
			},
			api.StatusFailure,
		},
		{
			aliveCtx,
			&api.URL{Scheme: "dummy-ping", Opaque: "degrade"},
			time.Now().Add(-20 * time.Second),
			pinger.Result{
				Target: &net.IPAddr{net.IPv4(127, 3, 2, 1), ""},
				Sent:   3,
				Recv:   2,
				Loss:   1,
				MinRTT: 1234 * time.Microsecond,
				AvgRTT: 2345 * time.Microsecond,
				MaxRTT: 3456 * time.Microsecond,
			},
			"some packets have dropped",
			map[string]interface{}{
				"rtt_min":      1.234,
				"rtt_avg":      2.345,
				"rtt_max":      3.456,
				"packets_recv": 2,
				"packets_sent": 3,
			},
			api.StatusDegrade,
		},
		{
			cancelCtx,
			&api.URL{Scheme: "dummy-ping", Opaque: "timeout"},
			time.Now().Add(-30 * time.Second),
			pinger.Result{
				Target: &net.IPAddr{net.IPv4(127, 3, 2, 1), ""},
				Sent:   3,
				Recv:   2,
				Loss:   1,
				MinRTT: 1234 * time.Microsecond,
				AvgRTT: 2345 * time.Microsecond,
				MaxRTT: 3456 * time.Microsecond,
			},
			"probe aborted",
			map[string]interface{}{
				"rtt_min":      1.234,
				"rtt_avg":      2.345,
				"rtt_max":      3.456,
				"packets_recv": 2,
				"packets_sent": 3,
			},
			api.StatusAborted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Target.String(), func(t *testing.T) {
			rec := pingResultToRecord(tt.Context, tt.Target, tt.StartTime, tt.Result)

			if !rec.Time.Equal(tt.StartTime) {
				t.Errorf("unexpected time: expected=%s actual=%s", tt.StartTime, rec.Time)
			}

			if rec.Status != tt.Status {
				t.Errorf("unexpected status: expected=%s actual=%s", tt.Status, rec.Status)
			}

			if rec.Latency != tt.Result.AvgRTT {
				t.Errorf("unexpected latency: expected=%s actual=%s", tt.Result.AvgRTT, rec.Latency)
			}

			if rec.Target.String() != tt.Target.String() {
				t.Errorf("unexpected target: expected=%s actual=%s", tt.Target, rec.Target)
			}

			if rec.Message != tt.Message {
				t.Errorf("unexpected message\n--- expected ---\n%s\n--- actual ---\n%s", tt.Message, rec.Message)
			}

			if diff := cmp.Diff(tt.Extra, rec.Extra); diff != "" {
				t.Errorf("unexpected extra\n%s", diff)
			}
		})
	}
}
