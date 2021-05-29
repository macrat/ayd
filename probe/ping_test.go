package probe_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/testutil"
)

func TestResourceLocker(t *testing.T) {
	rl := probe.NewResourceLocker()

	startCount := 0
	stopCount := 0

	start := func() {
		rl.Start(func() error {
			startCount++
			go rl.Teardown(func() {
				stopCount++
			})
			return nil
		})
	}
	stop := func() {
		rl.Done()
		time.Sleep(10 * time.Millisecond) // wait for teardown goroutine
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
}

func TestResourceLocker_flooding(t *testing.T) {
	rl := probe.NewResourceLocker()

	startCount := 0
	stopCount := 0

	start := func() {
		rl.Start(func() error {
			startCount++
			go rl.Teardown(func() {
				stopCount++
			})
			return nil
		})
	}

	for i := 0; i < 10000; i++ {
		start()
	}
	for i := 0; i < 10000; i++ {
		rl.Done()
	}

	time.Sleep(10 * time.Millisecond) // wait for teardown goroutine
	rl.Lock()
	defer rl.Unlock()

	if startCount != 1 {
		t.Errorf("unexpected start count: %d", startCount)
	}

	if stopCount != 1 {
		t.Errorf("unexpected stop count: %d", stopCount)
	}
}

func TestResourceLocker_goroutine_leak(t *testing.T) {
	rl := probe.NewResourceLocker()

	startCount := 0
	stopCount := 0

	start := func() {
		rl.Start(func() error {
			startCount++
			go rl.Teardown(func() {
				stopCount++
			})
			return nil
		})
	}

	before := runtime.NumGoroutine()
	for i := 0; i < 100000; i++ {
		start()
		rl.Done()
	}
	time.Sleep(10 * time.Millisecond)
	after := runtime.NumGoroutine()

	if before+10 < after {
		t.Errorf("number of goroutines is too increased: %d -> %d", before, after)
	}

	if startCount != stopCount {
		t.Errorf("miss match start count and stop count: stop=%d stop=%d", startCount, stopCount)
	}
}

func BenchmarkResourceLocker_Start(b *testing.B) {
	rl := probe.NewResourceLocker()

	starter := func() error {
		go rl.Teardown(func() {
		})
		return nil
	}

	for i := 0; i < b.N; i++ {
		rl.Start(starter)
	}
}

func BenchmarkResourceLocker_Done(b *testing.B) {
	rl := probe.NewResourceLocker()

	starter := func() error {
		go rl.Teardown(func() {
		})
		return nil
	}

	for i := 0; i < b.N; i++ {
		rl.Start(starter)
		rl.Done()
	}
}

func TestPingProbe(t *testing.T) {
	t.Parallel()

	if err := probe.CheckPingPermission(); err != nil {
		t.Fatalf("failed to check ping permission: %s", err)
	}

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", api.StatusHealthy, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping:127.0.0.1", api.StatusHealthy, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping:::1", api.StatusHealthy, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping:of-course-definitely-no-such-host", api.StatusUnknown, `.*`, ""},
	})

	t.Run("timeout", func(t *testing.T) {
		p := testutil.NewProbe(t, "ping:localhost")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		defer cancel()

		records := testutil.RunCheck(ctx, p)
		if len(records) != 1 {
			t.Fatalf("unexpected number of records: %#v", records)
		}

		if records[0].Status != api.StatusFailure {
			t.Errorf("unexpected status: %s", records[0].Status)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		p := testutil.NewProbe(t, "ping:localhost")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		records := testutil.RunCheck(ctx, p)
		if len(records) != 1 {
			t.Fatalf("unexpected number of records: %#v", records)
		}

		if records[0].Message != "probe aborted" {
			t.Errorf("unexpected message: %s", records[0].Message)
		}

		if records[0].Status != api.StatusAborted {
			t.Errorf("unexpected status: %s", records[0].Status)
		}
	})
}

func BenchmarkPingProbe(b *testing.B) {
	p := testutil.NewProbe(b, "ping:localhost")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
