package exporter_test

import (
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/exporter"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/store"
)

func TestLogScanner(t *testing.T) {
	tests := []struct {
		Name   string
		Output []string
		Since  time.Time
		Until  time.Time
	}{
		{
			"read_all",
			[]string{"first", "second", "last"},
			time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"drop_first",
			[]string{"second", "last"},
			time.Date(2000, 1, 1, 13, 2, 4, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"drop_last",
			[]string{"first", "second"},
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
		},
		{
			"drop_both",
			[]string{"second"},
			time.Date(2000, 1, 1, 13, 2, 4, 0, time.UTC),
			time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
		},
	}

	scanners := []struct {
		Name string
		F    func(since, until time.Time) exporter.LogScanner
	}{
		{
			"LogReader",
			func(since, until time.Time) exporter.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tfirst",
					"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tsecond",
					"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tlast",
				}, "\n")))

				return exporter.NewLogReaderFromReader(f, since, until)
			},
		},
		{
			"LogGenerator",
			func(since, until time.Time) exporter.LogScanner {
				s, err := store.New("")
				if err != nil {
					t.Fatalf("failed to create store: %s", err)
				}
				s.Console = io.Discard

				s.Report(api.Record{
					CheckedAt: time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
					Target:    &url.URL{Scheme: "dummy", Fragment: "hello"},
					Message:   "first",
				})
				s.Report(api.Record{
					CheckedAt: time.Date(2000, 1, 2, 13, 2, 3, 0, time.UTC),
					Target:    &url.URL{Scheme: "dummy", Fragment: "world"},
					Message:   "second",
				})
				s.Report(api.Record{
					CheckedAt: time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
					Target:    &url.URL{Scheme: "dummy", Fragment: "hello"},
					Message:   "last",
				})

				return exporter.NewLogGenerator(s, since, until)
			},
		},
	}

	for _, scanner := range scanners {
		t.Run(scanner.Name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.Name, func(t *testing.T) {
					r := scanner.F(tt.Since, tt.Until)
					defer r.Close()

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
		})
	}
}
