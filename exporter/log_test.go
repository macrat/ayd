package exporter_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/exporter"
)

func TestLogReader_Seek(t *testing.T) {
	tests := []struct {
		Name   string
		Input  []string
		Output []string
		Since  time.Time
		Until  time.Time
	}{
		{
			"empty",
			[]string{},
			[]string{},
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"read_all",
			[]string{
				"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tfirst",
				"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tsecond",
				"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tlast",
			},
			[]string{"first", "second", "last"},
			time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"drop_first",
			[]string{
				"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tfirst",
				"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tsecond",
				"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tlast",
			},
			[]string{"second", "last"},
			time.Date(2000, 1, 1, 13, 2, 4, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"drop_last",
			[]string{
				"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tfirst",
				"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tsecond",
				"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tlast",
			},
			[]string{"first", "second"},
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
		},
		{
			"drop_both",
			[]string{
				"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tfirst",
				"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tsecond",
				"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tlast",
			},
			[]string{"second"},
			time.Date(2000, 1, 1, 13, 2, 4, 0, time.UTC),
			time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			f := io.NopCloser(strings.NewReader(strings.Join(tt.Input, "\n")))

			r := exporter.NewLogReaderFromReader(f, tt.Since, tt.Until)

			var results []string
			for r.Scan() {
				results = append(results, r.Record().Message)
			}

			if len(results) != len(tt.Output) {
				t.Fatalf("unexpected number of output: %d\n%#v", len(results), results)
			}

			for i := range results {
				if results[i] != tt.Output[i] {
					t.Errorf("unexpected message at %d: %#v != %#v", i, results[i], tt.Output[i])
				}
			}
		})
	}
}
