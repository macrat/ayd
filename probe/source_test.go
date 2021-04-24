package probe_test

import (
	"context"
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
		{"source:./stub/healthy-list.txt", map[string]store.Status{
			"ping:127.0.0.1":                 store.STATUS_HEALTHY,
			"ping:localhost":                 store.STATUS_HEALTHY,
			"source:./stub/healthy-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:stub/healthy-list.txt", map[string]store.Status{
			"ping:127.0.0.1":               store.STATUS_HEALTHY,
			"ping:localhost":               store.STATUS_HEALTHY,
			"source:stub/healthy-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:./stub/failure-list.txt", map[string]store.Status{
			"ping:127.0.0.1":                 store.STATUS_HEALTHY,
			"ping:localhost":                 store.STATUS_HEALTHY,
			"tcp:localhost:56789":            store.STATUS_FAILURE,
			"source:./stub/failure-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:./stub/invalid-list.txt", map[string]store.Status{
			"source:./stub/invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: ::invalid host::, no-such-scheme:hello world, source:./stub/no-such-list.txt"},
		{"source:stub/invalid-list.txt", map[string]store.Status{
			"source:stub/invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: ::invalid host::, no-such-scheme:hello world, source:./stub/no-such-list.txt"},
		{"source:./stub/include-invalid-list.txt", map[string]store.Status{
			"source:./stub/include-invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: ::invalid host::, no-such-scheme:hello world, source:./stub/no-such-list.txt"},
		{"source:./stub/no-such-list.txt", map[string]store.Status{
			"source:./stub/no-such-list.txt": store.STATUS_UNKNOWN,
		}, `open \./stub/no-such-list\.txt: (no such file or directory|The system cannot find the file specified\.)`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Target, func(t *testing.T) {
			t.Parallel()

			p, err := probe.Get(tt.Target)
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

			rs := p.Check(ctx)

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
