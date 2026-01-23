package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestAydCommand_RunServer(t *testing.T) {
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

			cmd, _ := MakeTestCommand(t, tt.Args)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			code := cmd.RunServer(ctx, s)
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
	log, stdout := io.Pipe()
	defer log.Close()
	defer stdout.Close()
	s := testutil.NewStore(t, testutil.WithConsole(stdout))
	defer s.Close()

	cert := testutil.NewCertificate(t)
	cmd, _ := MakeTestCommand(t, []string{"dummy:"})
	cmd.CertPath = cert.CertFile
	cmd.KeyPath = cert.KeyFile

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		code := cmd.RunServer(ctx, s)
		if code != 0 {
			t.Errorf("unexpected return code: %d", code)
		}
		wg.Done()
	}()

	var startMessage struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(log).Decode(&startMessage); err != nil {
		t.Fatalf("failed to parse start message: %s", err)
	}

	go func() {
		// discard all outputs
		io.Copy(io.Discard, log)
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", startMessage.URL+"/status.html", nil)
	if err != nil {
		t.Fatalf("failed to make request: %s", err)
	}
	resp, err := cert.Client().Do(req)
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

	tests := []struct {
		Name      string
		Cert, Key string
		Pattern   string
		Code      int
	}{
		{"no-such-key", cert.CertFile, "./testdata/no-such-file.pem", "^error: key file does not exist: \\./testdata/no-such-file.pem\n$", 2},
		{"no-such-cert", "./testdata/no-such-file.pem", cert.KeyFile, "^error: certificate file does not exist: \\./testdata/no-such-file.pem\n$", 2},
		{"invalid-file", cert.KeyFile, cert.KeyFile, `{"time":"[^"]*", "status":"FAILURE", "latency":0.000, "target":"ayd:endpoint", "message":".*"}`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			cmd, output := MakeTestCommand(t, []string{"dummy:"})
			cmd.CertPath = tt.Cert
			cmd.KeyPath = tt.Key

			s := testutil.NewStore(t, testutil.WithConsole(output))

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			code := cmd.RunServer(ctx, s)
			if code != tt.Code {
				t.Errorf("unexpected return code: %d", code)
			}

			s.Close() // close store here for make sure that console log flushed

			if ok, _ := regexp.Match(tt.Pattern, output.Bytes()); !ok {
				t.Errorf("expected output matches with %q but does not match\n%s", tt.Pattern, output)
			}
		})
	}
}

func BenchmarkRunServer(b *testing.B) {
	s := testutil.NewStore(b)
	defer s.Close()

	tasks := make([]string, 1001)
	tasks[0] = "10ms"
	for i := range tasks[1:] {
		tasks[i+1] = fmt.Sprintf("dummy:#%d", i)
	}
	cmd, _ := MakeTestCommand(b, tasks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		cmd.RunServer(ctx, s)
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
