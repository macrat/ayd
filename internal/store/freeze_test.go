package store

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/store/freeze"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestFreezeProbeHistory(t *testing.T) {
	filledRecords := []api.Record{}
	for i := 0; i < PROBE_HISTORY_LEN-1; i++ {
		filledRecords = append(filledRecords, api.Record{
			CheckedAt: time.Date(2021, time.January, 2, 15, 4, 5, 0, time.UTC),
			Status:    api.StatusHealthy,
			Target:    &url.URL{Scheme: "ping", Opaque: "local"},
			Message:   "filled",
			Latency:   123456 * time.Microsecond,
		})
	}

	tests := []struct {
		Name        string
		Records     []api.Record
		Updated     string
		Status      string
		FirstRecord freeze.Record
		LastRecord  freeze.Record
	}{
		{
			Name:        "no-data",
			Records:     []api.Record{},
			Updated:     "",
			Status:      "NO_DATA",
			FirstRecord: freeze.Record{Status: "NO_DATA"},
			LastRecord:  freeze.Record{Status: "NO_DATA"},
		},
		{
			Name: "single-failure",
			Records: []api.Record{{
				CheckedAt: time.Date(2021, time.January, 2, 20, 1, 2, 0, time.UTC),
				Status:    api.StatusFailure,
				Latency:   654321 * time.Microsecond,
				Target:    &url.URL{Scheme: "ping", Opaque: "local"},
				Message:   "this is failure",
			}},
			Updated:     "2021-01-02T20:01:02Z",
			Status:      "FAILURE",
			FirstRecord: freeze.Record{Status: "NO_DATA"},
			LastRecord: freeze.Record{
				CheckedAt: "2021-01-02T20:01:02Z",
				Status:    "FAILURE",
				Latency:   654.321,
				Target:    "ping:local",
				Message:   "this is failure",
			},
		},
		{
			Name: "filled-unknown",
			Records: append(filledRecords, api.Record{
				CheckedAt: time.Date(2021, time.January, 2, 17, 4, 3, 0, time.UTC),
				Status:    api.StatusUnknown,
				Latency:   123321 * time.Microsecond,
				Message:   "this is unknown",
				Target:    &url.URL{Scheme: "ping", Opaque: "local"},
			}),
			Updated: "2021-01-02T17:04:03Z",
			Status:  "UNKNOWN",
			FirstRecord: freeze.Record{
				CheckedAt: "2021-01-02T15:04:05Z",
				Status:    "HEALTHY",
				Latency:   123.456,
				Target:    "ping:local",
				Message:   "filled",
			},
			LastRecord: freeze.Record{
				CheckedAt: "2021-01-02T17:04:03Z",
				Status:    "UNKNOWN",
				Latency:   123.321,
				Target:    "ping:local",
				Message:   "this is unknown",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			hs := &ProbeHistory{
				Target:  &url.URL{Scheme: "ping", Opaque: "localhost"},
				Records: tt.Records,
			}

			frozen := freezeProbeHistory(hs)

			if frozen.Target != "ping:localhost" {
				t.Errorf("unexpected target: %s", frozen.Target)
			}

			if frozen.Status != tt.Status {
				t.Errorf("unexpected status: %s", frozen.Status)
			}

			if frozen.Updated != tt.Updated {
				t.Errorf("unexpected updated: %s", frozen.Updated)
			}

			if len(frozen.History) != PROBE_HISTORY_LEN {
				t.Errorf("unexpected number of history: %d", len(frozen.History))
			}

			if !reflect.DeepEqual(frozen.History[0], tt.FirstRecord) {
				t.Errorf("unexpected first record: %#v", frozen.History[0])
			}

			if !reflect.DeepEqual(frozen.History[len(frozen.History)-1], tt.LastRecord) {
				t.Errorf("unexpected last record: %#v", frozen.History[len(frozen.History)-1])
			}
		})
	}
}

func BenchmarkFreeze(b *testing.B) {
	s, err := New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Freeze()
	}
}
