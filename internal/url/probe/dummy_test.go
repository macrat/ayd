package probe_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestDummyProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dummy:", api.StatusHealthy, ``, ""},
		{"dummy:healthy", api.StatusHealthy, ``, ""},
		{"dummy:debased", api.StatusDebased, ``, ""},
		{"dummy:failure", api.StatusFailure, ``, ""},
		{"dummy:aborted", api.StatusAborted, ``, ""},
		{"dummy:unknown", api.StatusUnknown, ``, ""},
		{"dummy:healthy?message=hello+world", api.StatusHealthy, `hello world`, ""},
		{"dummy:healthy#something-comment", api.StatusHealthy, ``, ""},

		{"dummy:unknown-status", api.StatusUnknown, ``, `opaque must healthy, debased, failure, aborted, unknown, or random`},
		{"dummy:?latency=1ms", api.StatusHealthy, ``, ""},
		{"dummy:?latency=1", api.StatusUnknown, ``, `time: missing unit in duration "1"`},
		{"dummy:?latency=1kb", api.StatusUnknown, ``, `time: unknown unit "kb" in duration "1kb"`},
	}, 1)

	t.Run("dummy:random", func(t *testing.T) {
		p := testutil.NewProbe(t, "dummy:random")

		h, d, f, u := 0, 0, 0, 0
		for i := 0; i < 800; i++ {
			rs := testutil.RunCheck(context.Background(), p)
			for _, r := range rs {
				switch r.Status {
				case api.StatusHealthy:
					h++
				case api.StatusDebased:
					d++
				case api.StatusFailure:
					f++
				case api.StatusUnknown:
					u++
				}
			}
		}

		if h < 150 || 250 < h {
			t.Errorf("number of healthy records was out of expected range: %d", h)
		}

		if d < 150 || 250 < d {
			t.Errorf("number of debased records was out of expected range: %d", f)
		}

		if f < 150 || 250 < f {
			t.Errorf("number of failure records was out of expected range: %d", f)
		}

		if u < 150 || 250 < u {
			t.Errorf("number of unknown records was out of expected range: %d", u)
		}
	})

	t.Run("dummy:healthy?latency=5s", func(t *testing.T) {
		p := testutil.NewProbe(t, "dummy:healthy?latency=5s")

		stime := time.Now()
		rs := testutil.RunCheck(context.Background(), p)
		latency := time.Now().Sub(stime)

		if latency < 4800*time.Millisecond || 5200*time.Millisecond < latency {
			t.Errorf("real latency was out of expected range: %s", latency)
		}

		for _, r := range rs {
			if r.Latency != 5*time.Second {
				t.Errorf("latency in record was unexpected value: %s", r.Latency)
			}
		}
	})

	t.Run("dummy:healthy?latency=5m/timeout", func(t *testing.T) {
		p := testutil.NewProbe(t, "dummy:healthy?latency=5m")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		stime := time.Now()
		rs := testutil.RunCheck(ctx, p)
		latency := time.Now().Sub(stime)

		if latency < 800*time.Millisecond || 1200*time.Millisecond < latency {
			t.Errorf("real latency was out of expected range: %s", latency)
		}

		for _, r := range rs {
			if r.Latency < 800*time.Millisecond || 1200*time.Millisecond < r.Latency {
				t.Errorf("latency in record was out of expected range: %s", r.Latency)
			}
			if r.Status != api.StatusFailure {
				t.Errorf("unexpected status: %s", r.Status)
			}
			if r.Message != "probe timed out" {
				t.Errorf("unexpected message: %#v", r.Message)
			}
		}
	})

	AssertTimeout(t, "dummy:")
}
