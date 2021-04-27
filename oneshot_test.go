package main_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
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
		{[]string{"dummy:?latency=200ms"}, 1, 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.Args), func(t *testing.T) {
			f, err := os.CreateTemp("", "ayd-test-*")
			if err != nil {
				t.Fatalf("failed to create log file: %s", err)
			}
			defer os.Remove(f.Name())
			f.Close()

			s, err := store.New(f.Name())
			if err != nil {
				t.Fatalf("failed to create store: %s", err)
			}
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

func BenchmarkRunOneshot(b *testing.B) {
	for _, status := range []store.Status{store.STATUS_UNKNOWN, store.STATUS_HEALTHY, store.STATUS_FAILURE} {
		name := status.String()
		if status == store.STATUS_UNKNOWN {
			name = "RANDOM"
		}

		b.Run(name, func(b *testing.B) {
			for _, n := range []int{10, 25, 50, 75, 100, 250, 500, 750, 1000} {
				b.Run(fmt.Sprintf("%dtargets", n), func(b *testing.B) {
					f, err := os.CreateTemp("", "ayd-test-*")
					if err != nil {
						b.Fatalf("failed to create log file: %s", err)
					}
					defer os.Remove(f.Name())
					f.Close()

					s, err := store.New(f.Name())
					if err != nil {
						b.Fatalf("failed to create store: %s", err)
					}
					s.Console = io.Discard
					defer s.Close()

					tasks := make([]main.Task, n)
					schedule, _ := main.ParseIntervalSchedule("1s")
					for i := range tasks {
						p, err := probe.New(fmt.Sprintf("dummy:random#benchmark-%d", i))
						if err != nil {
							b.Fatalf("failed to create probe: %s", err)
						}
						tasks[i] = main.Task{Schedule: schedule, Probe: p}
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
