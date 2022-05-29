package scheme_test

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestPingProbe_Probe(t *testing.T) {
	t.Parallel()

	if _, err := scheme.NewPingProbe(&api.URL{Scheme: "ping", Opaque: "localhost"}); err != nil {
		t.Fatalf("failed to check ping permission: %s", err)
	}

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", api.StatusHealthy, `All packets came back`, ""},
		{"ping:127.0.0.1", api.StatusHealthy, `All packets came back`, ""},
		{"ping:::1", api.StatusHealthy, `All packets came back`, ""},
		{"ping4:localhost", api.StatusHealthy, `All packets came back`, ""},
		{"ping6:localhost", api.StatusHealthy, `All packets came back`, ""},
		{"ping:of-course-definitely-no-such-host", api.StatusUnknown, `.*`, ""},
	}, 2)

	t.Run("timeout", func(t *testing.T) {
		p := testutil.NewProber(t, "ping:localhost")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
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

		AssertProbe(t, []ProbeTest{
			{"ping:localhost", api.StatusHealthy, `All packets came back`, ""},
		}, 2)
	})
}

func TestPingProbe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unprivileged mode is not supported on windows")
	}

	privileged := os.Getenv("AYD_PRIVILEGED")
	t.Cleanup(func() {
		os.Setenv("AYD_PRIVILEGED", privileged)
	})

	tests := []struct {
		Env  string
		Fail bool
	}{
		{"1", true},
		{"0", false},
		{"yes", true},
		{"no", false},
		{"true", true},
		{"false", false},
		{"TRUE", true},
		{"False", false},
	}

	for _, tt := range tests {
		t.Run("AYD_PRIVILEGED="+tt.Env, func(t *testing.T) {
			os.Setenv("AYD_PRIVILEGED", tt.Env)
			_, err := scheme.NewPingProbe(&api.URL{Scheme: "ping", Opaque: "localhost"})

			if tt.Fail && !errors.Is(err, scheme.ErrFailedToPreparePing) {
				t.Errorf("expected permission error but got %v", err)
			}
			if !tt.Fail && err != nil {
				t.Errorf("expected nil but got error: %s", err)
			}
		})
	}
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
