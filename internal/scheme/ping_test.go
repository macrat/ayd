//go:build !githubci || !linux

package scheme_test

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestPingProbe_Probe(t *testing.T) {
	if _, err := scheme.NewPingProbe(&api.URL{Scheme: "ping", Opaque: "localhost"}); err != nil {
		t.Fatalf("failed to check ping permission: %s", err)
	}

	pattern := strings.Join([]string{
		`all packets came back`,
		`---`,
		`packets_recv: 3`,
		`packets_sent: 3`,
		`rtt_avg: [0-9]+(\.[0-9]+)?`,
		`rtt_max: [0-9]+(\.[0-9]+)?`,
		`rtt_min: [0-9]+(\.[0-9]+)?`,
	}, "\n")

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", api.StatusHealthy, pattern, ""},
		{"ping:127.0.0.1", api.StatusHealthy, pattern, ""},
		{"ping:::1", api.StatusHealthy, pattern, ""},
		{"ping4:localhost", api.StatusHealthy, pattern, ""},
		{"ping6:localhost", api.StatusHealthy, pattern, ""},
		{"ping:of-course-definitely-no-such-host", api.StatusUnknown, "[^\n]*", ""},
	}, 2)

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()

		p := testutil.NewProber(t, "ping:localhost")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		if runtime.GOOS == "windows" {
			// Windows on GitHub Actions is incredible slow so I need more time to make sure that completely timed out
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(10 * time.Millisecond)
		}
		defer cancel()

		records := testutil.RunProbe(ctx, p)
		if len(records) != 1 {
			t.Fatalf("unexpected number of records: %#v", records)
		}

		if records[0].Status != api.StatusFailure {
			t.Errorf("unexpected status: %s", records[0].Status)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		t.Parallel()

		p := testutil.NewProber(t, "ping:localhost")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		records := testutil.RunProbe(ctx, p)
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

	t.Run("with-settings", func(t *testing.T) {
		t.Setenv("AYD_PING_PACKETS", "10")
		t.Setenv("AYD_PING_INTERVAL", "1ms")

		pattern := strings.Join([]string{
			`all packets came back`,
			`---`,
			`packets_recv: 10`,
			`packets_sent: 10`,
			`rtt_avg: [0-9]+(\.[0-9]+)?`,
			`rtt_max: [0-9]+(\.[0-9]+)?`,
			`rtt_min: [0-9]+(\.[0-9]+)?`,
		}, "\n")

		AssertProbe(t, []ProbeTest{
			{"ping:localhost", api.StatusHealthy, pattern, ""},
		}, 2)
	})
}

func BenchmarkPingProbe(b *testing.B) {
	p := testutil.NewProber(b, "ping:localhost")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Probe(ctx, r)
	}
}
