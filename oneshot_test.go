package main_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/macrat/ayd"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/testutil"
)

func TestRunOneshot(t *testing.T) {
	tests := []struct {
		Args    []string
		Records int
		Code    int
	}{
		{[]string{"dummy:#with-healthy", "dummy:healthy", "dummy:"}, 3, 0},
		{[]string{"dummy:#with-failure", "dummy:failure", "dummy:"}, 3, 1},
		{[]string{"dummy:#with-unknown", "dummy:unknown", "dummy:"}, 3, 2},
		{[]string{"dummy:#with-interval", "10m", "dummy:healthy"}, 2, 0},
		{[]string{"dummy:#single-target"}, 1, 0},
		{[]string{"dummy:?latency=10ms"}, 1, 0},
		{[]string{"dummy:?latency=200ms"}, 1, 1},
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

			code := main.RunOneshot(ctx, s, tasks)
			if code != tt.Code {
				t.Errorf("unexpected exit code: %d", code)
			}

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

func BenchmarkRunOneshot(b *testing.B) {
	for _, status := range []api.Status{api.StatusUnknown, api.StatusHealthy, api.StatusFailure} {
		name := status.String()
		if status == api.StatusUnknown {
			name = "RANDOM"
		}

		b.Run(name, func(b *testing.B) {
			for _, n := range []int{10, 25, 50, 75, 100, 250, 500, 750, 1000} {
				b.Run(fmt.Sprintf("%dtargets", n), func(b *testing.B) {
					s := testutil.NewStore(b)
					defer s.Close()

					tasks := make([]main.Task, n)
					schedule, _ := main.ParseIntervalSchedule("1s")
					for i := range tasks {
						tasks[i] = main.Task{
							Schedule: schedule,
							Probe:    testutil.NewProbe(b, fmt.Sprintf("dummy:random#benchmark-%d", i)),
						}
					}

					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						main.RunOneshot(ctx, s, tasks)
					}
				})
			}
		})
	}
}
