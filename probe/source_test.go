package probe_test

import (
	"testing"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

func TestSource(t *testing.T) {
	tests := []struct {
		Target  string
		Records map[string]store.Status
		Error   string
	}{
		{"source:./stub/healthy-list.txt", map[string]store.Status{
			"ping:127.0.0.1":                 store.STATUS_HEALTHY,
			"ping:localhost":                 store.STATUS_HEALTHY,
			"source:./stub/healthy-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:./stub/failure-list.txt", map[string]store.Status{
			"ping:127.0.0.1":                 store.STATUS_HEALTHY,
			"ping:localhost":                 store.STATUS_HEALTHY,
			"tcp:localhost:56789":            store.STATUS_FAILURE,
			"source:./stub/failure-list.txt": store.STATUS_HEALTHY,
		}, ""},
		{"source:./stub/invalid-list.txt", map[string]store.Status{
			"source:./stub/invalid-list.txt": store.STATUS_UNKNOWN,
		}, "Invalid URI: invalid host"},
		{"source:./stub/no-such-list.txt", map[string]store.Status{
			"source:./stub/no-such-list.txt": store.STATUS_UNKNOWN,
		}, "open ./stub/no-such-list.txt: no such file or directory"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Target, func(t *testing.T) {
			t.Parallel()

			p, err := probe.Get(tt.Target)
			if err != nil && tt.Error == "" {
				t.Fatalf("failed to create probe: %s", err)
			}
			if tt.Error != "" {
				if err == nil {
					t.Fatalf("expected error %#v but got nil", tt.Error)
				} else if err.Error() != tt.Error {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			rs := p.Check()

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
