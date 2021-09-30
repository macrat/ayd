package probe_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/testutil"
)

func TestSource(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	server := RunDummyHTTPServer()
	defer server.Close()

	tests := []struct {
		Target       string
		Records      map[string]api.Status
		ErrorPattern string
	}{
		{"source:./testdata/healthy-list.txt", map[string]api.Status{
			"dummy:healthy#sub-list":             api.StatusHealthy,
			"dummy:healthy#healthy-list":         api.StatusHealthy,
			"source:./testdata/healthy-list.txt": api.StatusHealthy,
		}, ""},
		{"source:testdata/healthy-list.txt", map[string]api.Status{
			"dummy:healthy#sub-list":           api.StatusHealthy,
			"dummy:healthy#healthy-list":       api.StatusHealthy,
			"source:testdata/healthy-list.txt": api.StatusHealthy,
		}, ""},
		{"source:./testdata/failure-list.txt", map[string]api.Status{
			"dummy:healthy#sub-list":             api.StatusHealthy,
			"dummy:healthy#failure-list":         api.StatusHealthy,
			"dummy:failure":                      api.StatusFailure,
			"source:./testdata/failure-list.txt": api.StatusHealthy,
		}, ""},
		{"source:./testdata/invalid-list.txt", map[string]api.Status{
			"source:./testdata/invalid-list.txt": api.StatusUnknown,
		}, "invalid source: invalid URL: ::invalid host::, no-such-scheme:hello world, source:./testdata/no-such-list.txt"},
		{"source:testdata/invalid-list.txt", map[string]api.Status{
			"source:testdata/invalid-list.txt": api.StatusUnknown,
		}, "invalid source: invalid URL: ::invalid host::, no-such-scheme:hello world, source:./testdata/no-such-list.txt"},
		{"source:./testdata/include-invalid-list.txt", map[string]api.Status{
			"source:./testdata/include-invalid-list.txt": api.StatusUnknown,
		}, "invalid source: invalid URL: ::invalid host::, no-such-scheme:hello world, source:./testdata/no-such-list.txt"},
		{"source:./testdata/no-such-list.txt", map[string]api.Status{
			"source:./testdata/no-such-list.txt": api.StatusUnknown,
		}, `invalid source: open \./testdata/no-such-list\.txt: (no such file or directory|The system cannot find the file specified\.)`},
		{"source:" + path.Join(cwd, "testdata/sub-list.txt"), map[string]api.Status{
			"dummy:healthy#sub-list":                            api.StatusHealthy,
			"source:" + path.Join(cwd, "testdata/sub-list.txt"): api.StatusHealthy,
		}, ""},
		{"source:" + path.Join(cwd, "testdata/sub-list.txt"), map[string]api.Status{
			"dummy:healthy#sub-list":                            api.StatusHealthy,
			"source:" + path.Join(cwd, "testdata/sub-list.txt"): api.StatusHealthy,
		}, ""},
		{"source+" + server.URL + "/source", map[string]api.Status{
			"dummy:healthy":                    api.StatusHealthy,
			"ping:localhost":                   api.StatusHealthy,
			"source+" + server.URL + "/source": api.StatusHealthy,
		}, ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Target, func(t *testing.T) {
			p, err := probe.New(tt.Target)
			if err != nil && tt.ErrorPattern == "" {
				t.Fatalf("failed to create probe: %s", err)
			}
			if tt.ErrorPattern != "" {
				if err == nil {
					t.Fatalf("expected error %v but got nil", tt.ErrorPattern)
				} else if ok, _ := regexp.MatchString("^"+tt.ErrorPattern+"$", err.Error()); !ok {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			rs := testutil.RunCheck(ctx, p)

			if len(rs) != len(tt.Records) {
				t.Fatalf("unexpected number of records: %d\n%v", len(rs), rs)
			}

			for _, r := range rs {
				expect, ok := tt.Records[r.Target.String()]
				if !ok {
					t.Errorf("got unexpected or duplicated record: %s", r.Target)
					continue
				}
				if r.Status != expect {
					t.Errorf("got unexpected status: %s: expected %s but got %s", r.Target, expect, r.Status)
				}
				delete(tt.Records, r.Target.String())
			}

			for target := range tt.Records {
				t.Errorf("missing record of %s", target)
			}
		})
	}
}

func BenchmarkSource_load(b *testing.B) {
	for _, n := range []int{10, 25, 50, 75, 100, 250, 500, 750, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f, err := os.CreateTemp("", "ayd-test-*-list.txt")
			if err != nil {
				b.Fatalf("failed to create test file: %s", err)
			}
			defer f.Close()
			defer os.Remove(f.Name())

			for i := 0; i < n; i++ {
				fmt.Fprintf(f, "ping:host-%d\n", i)
			}

			target := &url.URL{Scheme: "source", Opaque: f.Name()}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = probe.NewSourceProbe(target)
			}
		})
	}
}

func BenchmarkSource(b *testing.B) {
	for _, n := range []int{10, 25, 50, 75, 100, 250, 500, 750, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f, err := os.CreateTemp("", "ayd-test-*-list.txt")
			if err != nil {
				b.Fatalf("failed to create test file: %s", err)
			}
			defer f.Close()
			defer os.Remove(f.Name())

			for i := 0; i < n; i++ {
				fmt.Fprintf(f, "dummy:healthy#%d\n", i)
			}

			target := &url.URL{Scheme: "source", Opaque: f.Name()}

			p, err := probe.NewSourceProbe(target)
			if err != nil {
				b.Fatalf("failed to create probe: %s", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			r := &testutil.DummyReporter{}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Check(ctx, r)
			}
		})
	}
}
