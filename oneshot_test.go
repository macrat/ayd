package main_test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/store"
)

func TestRunOneshot(t *testing.T) {
	tests := []struct {
		Args    []string
		Records int
		Code    int
	}{
		{[]string{"exec:echo#with-healthy", "exec:echo#::status::healthy", "exec:echo#hello"}, 3, 0},
		{[]string{"exec:echo#with-failure", "exec:echo#::status::failure", "exec:echo#hello"}, 3, 1},
		{[]string{"exec:echo#with-unknown", "exec:echo#::status::unknown", "exec:echo#hello"}, 3, 2},
		{[]string{"exec:echo#with-interval", "10m", "exec:echo#hello"}, 2, 0},
		{[]string{"exec:echo#single-target"}, 1, 0},
		{[]string{"exec:sleep#0.01"}, 1, 0},
		{[]string{"exec:sleep#0.2"}, 1, 2},
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
			s.Lock()
			defer s.Unlock()

			count := 0
			for _, xs := range s.ProbeHistory {
				count += len(xs.Records)
			}

			if count != tt.Records {
				t.Errorf("unexpected number of probe history: %d", count)
			}
		})
	}
}

type DummyProbe struct {
	target *url.URL
	status store.Status
}

func (p DummyProbe) Target() *url.URL {
	return p.target
}

func (p DummyProbe) Check(ctx context.Context) []store.Record {
	status := p.status
	if p.status == store.STATUS_UNKNOWN {
		status = []store.Status{store.STATUS_UNKNOWN, store.STATUS_HEALTHY, store.STATUS_FAILURE}[rand.Intn(3)]
	}
	return []store.Record{{
		Target:  p.target,
		Status:  status,
		Message: p.target.Opaque,
	}}
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
						tasks[i] = main.Task{Schedule: schedule, Probe: DummyProbe{target: &url.URL{Scheme: "dummy", Opaque: fmt.Sprintf("benchmark-%d", i)}, status: status}}
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
