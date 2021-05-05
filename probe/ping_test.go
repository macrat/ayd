package probe_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestPingProbe(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := probe.StartPinger(ctx); err != nil {
		t.Fatalf("failed to start pinger: %s", err)
	}

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=4/4`, ""},
		{"ping:127.0.0.1", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=4/4`, ""},
		{"ping:::1", store.STATUS_HEALTHY, `rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=4/4`, ""},
		{"ping:of-course-definitely-no-such-host", store.STATUS_UNKNOWN, `.*`, ""},
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

		if records[0].Status != store.STATUS_FAILURE {
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

		if records[0].Status != store.STATUS_ABORTED {
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
