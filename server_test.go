package main_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestRunServer(t *testing.T) {
	tests := []struct {
		Args    []string
		Records int
	}{
		{[]string{"dummy:#with-healthy", "dummy:healthy", "dummy:"}, 3},
		{[]string{"dummy:#with-failure", "dummy:failure", "dummy:"}, 3},
		{[]string{"dummy:#with-unknown", "dummy:unknown", "dummy:"}, 3},
		{[]string{"dummy:#with-interval", "10m", "dummy:"}, 2},
		{[]string{"dummy:#single-target"}, 1},
		{[]string{"dummy:?latency=10ms"}, 1},
		{[]string{"dummy:?latency=200ms"}, 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.Args), func(t *testing.T) {
			s := testutil.NewStore(t)
			defer s.Close()

			tasks, errs := main.ParseArgs(tt.Args)
			if errs != nil {
				t.Fatalf("unexpected errors: %s", errs)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			code := main.RunServer(ctx, s, tasks)
			if code != 0 {
				t.Errorf("unexpected exit code: %d", code)
			}

			time.Sleep(10 * time.Millisecond)

			count := 0
			for _, xs := range s.ProbeHistory() {
				t.Log(len(xs.Records), "records by", xs.Target)
				count += len(xs.Records)
			}

			if count != tt.Records {
				t.Errorf("unexpected number of probe history: %d", count)
			}
		})
	}
}

func BenchmarkRunServer(b *testing.B) {
	s := testutil.NewStore(b)
	defer s.Close()

	schedule, _ := main.ParseIntervalSchedule("10ms")
	tasks := make([]main.Task, 1000)
	for i := range tasks {
		tasks[i].Schedule = schedule
		tasks[i].Probe = testutil.NewProbe(b, fmt.Sprintf("dummy:#%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		main.RunServer(ctx, s, tasks)
		cancel()
	}
	b.StopTimer()

	done := 0
	timeout := 0

	for _, x := range s.ProbeHistory() {
		for _, r := range x.Records {
			if r.Status == store.STATUS_HEALTHY {
				done++
			} else {
				timeout++
			}
		}
	}

	b.ReportMetric(float64(done)/float64(b.N), "done/op")
	b.ReportMetric(float64(timeout)/float64(b.N), "timeout/op")
	b.ReportMetric(1000*10-float64(done+timeout)/float64(b.N), "not-scheduled/op")
}
