package probe_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

func TestSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Target       string
		Records      map[string]store.Status
		ErrorPattern string
	}{
		{"source:./testdata/healthy-list.txt", map[string]store.Status{
			"dummy:healthy#sub-list":             store.STATUS_HEALTHY,
			"dummy:healthy#healthy-list":         store.STATUS_HEALTHY,
			"source:./testdata/healthy-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:testdata/healthy-list.txt", map[string]store.Status{
			"dummy:healthy#sub-list":           store.STATUS_HEALTHY,
			"dummy:healthy#healthy-list":       store.STATUS_HEALTHY,
			"source:testdata/healthy-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:./testdata/failure-list.txt", map[string]store.Status{
			"dummy:healthy#sub-list":             store.STATUS_HEALTHY,
			"dummy:healthy#failure-list":         store.STATUS_HEALTHY,
			"dummy:failure":                      store.STATUS_FAILURE,
			"source:./testdata/failure-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:./testdata/invalid-list.txt", map[string]store.Status{
			"source:./testdata/invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: ::invalid host::, no-such-scheme:hello world, source:./testdata/no-such-list.txt"},
		{"source:testdata/invalid-list.txt", map[string]store.Status{
			"source:testdata/invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: ::invalid host::, no-such-scheme:hello world, source:./testdata/no-such-list.txt"},
		{"source:./testdata/include-invalid-list.txt", map[string]store.Status{
			"source:./testdata/include-invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: ::invalid host::, no-such-scheme:hello world, source:./testdata/no-such-list.txt"},
		{"source:./testdata/no-such-list.txt", map[string]store.Status{
			"source:./testdata/no-such-list.txt": store.STATUS_UNKNOWN,
		}, `open \./testdata/no-such-list\.txt: (no such file or directory|The system cannot find the file specified\.)`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Target, func(t *testing.T) {
			t.Parallel()

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

			rs := []store.Record{}
			p.Check(ctx, (*DummyReporter)(&rs))

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
