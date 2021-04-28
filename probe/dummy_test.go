package probe_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

func TestDummyProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dummy:", store.STATUS_HEALTHY, ``},
		{"dummy:healthy", store.STATUS_HEALTHY, ``},
		{"dummy:failure", store.STATUS_FAILURE, ``},
		{"dummy:unknown", store.STATUS_UNKNOWN, ``},
		{"dummy:healthy?message=hello+world", store.STATUS_HEALTHY, `hello world`},
		{"dummy:healthy#something-comment", store.STATUS_HEALTHY, ``},
	})

	t.Run("dummy:random", func(t *testing.T) {
		p, err := probe.New("dummy:random")
		if err != nil {
			t.Fatalf("failed to create probe: %s", err)
		}

		h, f, u := 0, 0, 0
		for i := 0; i < 600; i++ {
			rs := []store.Record{}
			p.Check(context.Background(), (*DummyReporter)(&rs))
			for _, r := range rs {
				switch r.Status {
				case store.STATUS_HEALTHY:
					h++
				case store.STATUS_FAILURE:
					f++
				case store.STATUS_UNKNOWN:
					u++
				}
			}
		}

		if h < 150 || 250 < h {
			t.Errorf("number of healthy records was out of expected range: %d", h)
		}

		if f < 150 || 250 < f {
			t.Errorf("number of failure records was out of expected range: %d", f)
		}

		if u < 150 || 250 < u {
			t.Errorf("number of unknown records was out of expected range: %d", u)
		}
	})

	t.Run("dummy:healthy?latency=5s", func(t *testing.T) {
		p, err := probe.New("dummy:healthy?latency=5s")
		if err != nil {
			t.Fatalf("failed to create probe: %s", err)
		}

		stime := time.Now()
		rs := []store.Record{}
		p.Check(context.Background(), (*DummyReporter)(&rs))
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
		p, err := probe.New("dummy:healthy?latency=5m")
		if err != nil {
			t.Fatalf("failed to create probe: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		stime := time.Now()
		rs := []store.Record{}
		p.Check(ctx, (*DummyReporter)(&rs))
		latency := time.Now().Sub(stime)

		if latency < 800*time.Millisecond || 1200*time.Millisecond < latency {
			t.Errorf("real latency was out of expected range: %s", latency)
		}

		for _, r := range rs {
			if r.Latency < 800*time.Millisecond || 1200*time.Millisecond < r.Latency {
				t.Errorf("latency in record was out of expected range: %s", r.Latency)
			}
			if r.Status != store.STATUS_UNKNOWN {
				t.Errorf("unexpected status: %s", r.Status)
			}
			if r.Message != "probe timed out" {
				t.Errorf("unexpected message: %#v", r.Message)
			}
		}
	})

	AssertTimeout(t, "dummy:")
}
