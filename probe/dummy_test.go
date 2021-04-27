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
			for _, r := range p.Check(context.Background()) {
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

		if h < 180 || 220 < h {
			t.Errorf("number of healthy records was out of expected range: %d", h)
		}

		if f < 180 || 220 < f {
			t.Errorf("number of failure records was out of expected range: %d", f)
		}

		if u < 180 || 220 < u {
			t.Errorf("number of unknown records was out of expected range: %d", u)
		}
	})

	t.Run("dummy:healthy?latency=5s", func(t *testing.T) {
		p, err := probe.New("dummy:healthy?latency=5s")
		if err != nil {
			t.Fatalf("failed to create probe: %s", err)
		}

		stime := time.Now()
		rs := p.Check(context.Background())
		latency := time.Now().Sub(stime)

		if latency < 4100*time.Millisecond || 5100*time.Millisecond < latency {
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
		rs := p.Check(ctx)
		latency := time.Now().Sub(stime)

		if latency < 900*time.Millisecond || 1100*time.Millisecond < latency {
			t.Errorf("real latency was out of expected range: %s", latency)
		}

		for _, r := range rs {
			if r.Latency < 900*time.Millisecond || 1100*time.Millisecond < r.Latency {
				t.Errorf("latency in record was out of expected range: %s", r.Latency)
			}
			if r.Message != "timed out or interrupted" {
				t.Errorf("unexpected message: %#v", r.Message)
			}
		}
	})
}
