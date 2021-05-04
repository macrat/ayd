package exporter

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/macrat/ayd/store"
)

func TestFreezeProbeHistory(t *testing.T) {
	filledRecords := []*store.Record{}
	for i := 0; i < store.PROBE_HISTORY_LEN-1; i++ {
		filledRecords = append(filledRecords, &store.Record{
			CheckedAt: time.Date(2021, time.January, 2, 15, 4, 5, 0, time.UTC),
			Status:    store.STATUS_HEALTHY,
			Target:    &url.URL{Scheme: "ping", Opaque: "local"},
			Message:   "filled",
			Latency:   123456 * time.Microsecond,
		})
	}

	tests := []struct {
		Name        string
		Records     []*store.Record
		Updated     string
		Status      string
		FirstRecord frozenRecord
		LastRecord  frozenRecord
	}{
		{
			Name:        "no-data",
			Records:     []*store.Record{},
			Updated:     "",
			Status:      "NO_DATA",
			FirstRecord: frozenRecord{Status: "NO_DATA"},
			LastRecord:  frozenRecord{Status: "NO_DATA"},
		},
		{
			Name: "single-failure",
			Records: []*store.Record{&store.Record{
				CheckedAt: time.Date(2021, time.January, 2, 20, 1, 2, 0, time.UTC),
				Target:    &url.URL{Scheme: "ping", Opaque: "local"},
				Status:    store.STATUS_FAILURE,
				Message:   "this is failure",
				Latency:   654321 * time.Microsecond,
			}},
			Updated:     "2021-01-02T20:01:02Z",
			Status:      "FAILURE",
			FirstRecord: frozenRecord{Status: "NO_DATA"},
			LastRecord: frozenRecord{
				CheckedAt:  "2021-01-02T20:01:02Z",
				Status:     "FAILURE",
				Message:    "this is failure",
				Latency:    654.321,
				LatencyStr: "654.321ms",
			},
		},
		{
			Name: "filled-unknown",
			Records: append(filledRecords, &store.Record{
				CheckedAt: time.Date(2021, time.January, 2, 17, 4, 3, 0, time.UTC),
				Target:    &url.URL{Scheme: "ping", Opaque: "local"},
				Status:    store.STATUS_UNKNOWN,
				Message:   "this is unknown",
				Latency:   123321 * time.Microsecond,
			}),
			Updated: "2021-01-02T17:04:03Z",
			Status:  "UNKNOWN",
			FirstRecord: frozenRecord{
				CheckedAt:  "2021-01-02T15:04:05Z",
				Status:     "HEALTHY",
				Message:    "filled",
				Latency:    123.456,
				LatencyStr: "123.456ms",
			},
			LastRecord: frozenRecord{
				CheckedAt:  "2021-01-02T17:04:03Z",
				Status:     "UNKNOWN",
				Message:    "this is unknown",
				Latency:    123.321,
				LatencyStr: "123.321ms",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			hs := &store.ProbeHistory{
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

			if len(frozen.History) != store.PROBE_HISTORY_LEN {
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
	s, err := store.New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		freezeStatus(s)
	}
}
