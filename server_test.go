package main_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
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

			code := main.RunServer(ctx, s, tasks, "", "")
			if code != 0 {
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

func TestRunServer_tls(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	tasks, errs := main.ParseArgs([]string{"dummy:"})
	if errs != nil {
		t.Fatalf("failed to parse argument:\n%s", errs)
	}

	cert := testutil.NewCertificate(t)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		code := main.RunServer(ctx, s, tasks, cert.CertFile, cert.KeyFile)
		if code != 0 {
			t.Errorf("unexpected return code: %d", code)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond) // wait for start HTTP server

	resp, err := cert.Client().Get("https://localhost:9000/status.html")
	if err != nil {
		t.Fatalf("failed to fetch status page: %s", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("unexpected response status: %s", resp.Status)
	}

	cancel()
	wg.Wait()
}

func TestRunServer_tls_error(t *testing.T) {
	cert := testutil.NewCertificate(t)

	tasks, errs := main.ParseArgs([]string{"dummy:"})
	if errs != nil {
		t.Fatalf("failed to parse argument:\n%s", errs)
	}

	tests := []struct {
		Name      string
		Cert, Key string
		Code      int
	}{
		{"no-such-key", cert.CertFile, "./testdata/no-such-file.pem", 2},
		{"no-such-cert", "./testdata/no-such-file.pem", cert.KeyFile, 2},
		{"invalid-file", cert.KeyFile, cert.CertFile, 1},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s := testutil.NewStore(t)
			defer s.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			code := main.RunServer(ctx, s, tasks, tt.Cert, tt.Key)
			if code != tt.Code {
				t.Errorf("unexpected return code: %d", code)
			}
		})
	}
}

func TestRunServer_permissionError(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("permission test only works on *nix OS")
	}

	s := testutil.NewStore(t)
	defer s.Close()
	os.Chmod(s.Path, 0200)

	tasks, errs := main.ParseArgs([]string{"dummy:"})
	if errs != nil {
		t.Fatalf("failed to parse argument:\n%s", errs)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	code := main.RunServer(ctx, s, tasks, "", "")
	if code != 1 {
		t.Errorf("unexpected return code: %d", code)
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
		main.RunServer(ctx, s, tasks, "", "")
		cancel()
	}
	b.StopTimer()

	done := 0
	timeout := 0

	for _, x := range s.ProbeHistory() {
		for _, r := range x.Records {
			if r.Status == api.StatusHealthy {
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
