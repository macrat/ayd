package scheme_test

import (
	"context"
	"errors"
	"net/url"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestPingScheme_Probe(t *testing.T) {
	t.Parallel()

	if err := scheme.CheckPingPermission(); err != nil {
		t.Fatalf("failed to check ping permission: %s", err)
	}

	AssertProbe(t, []ProbeTest{
		{"ping:localhost", api.StatusHealthy, `ip=(127.0.0.1|::1) rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping:127.0.0.1", api.StatusHealthy, `ip=127.0.0.1 rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping:::1", api.StatusHealthy, `ip=::1 rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping4:localhost", api.StatusHealthy, `ip=127.0.0.1 rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
		{"ping6:localhost", api.StatusHealthy, `ip=::1 rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
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
}

func TestPingScheme_privilegedEnv(t *testing.T) {
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
			_, err := scheme.NewPingScheme(&url.URL{Scheme: "ping", Opaque: "localhost"})

			if tt.Fail && !errors.Is(err, scheme.ErrFailedToPreparePing) {
				t.Errorf("expected permission error but got %v", err)
			}
			if !tt.Fail && err != nil {
				t.Errorf("expected nil but got error: %s", err)
			}
		})
	}
}

func TestPingScheme_Alert(t *testing.T) {
	t.Parallel()

	if err := scheme.CheckPingPermission(); err != nil {
		t.Fatalf("failed to check ping permission: %s", err)
	}

	AssertAlert(t, []ProbeTest{
		{"ping:localhost", api.StatusHealthy, `ip=(127.0.0.1|::1) rtt\(min/avg/max\)=[0-9.]*/[0-9.]*/[0-9.]* send/recv=3/3`, ""},
	}, 5)
}

func BenchmarkPingScheme(b *testing.B) {
	p := testutil.NewProber(b, "ping:localhost")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Probe(ctx, r)
	}
}
