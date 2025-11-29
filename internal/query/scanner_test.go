package query_test

import (
	"testing"
	"time"

	"github.com/macrat/ayd/internal/query"
	api "github.com/macrat/ayd/lib-ayd"
)

// mockLogScanner implements api.LogScanner for testing.
type mockLogScanner struct {
	logs   []api.Record
	index  int
	closed bool
}

func (s *mockLogScanner) Scan() bool {
	if s.index < len(s.logs) {
		s.index++
		return true
	}
	return false
}

func (s *mockLogScanner) Record() api.Record {
	return s.logs[s.index-1]
}

func (s *mockLogScanner) Close() error {
	s.closed = true
	return nil
}

func TestFilterScanner_Scan(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	logs := []api.Record{
		{
			Time:   time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
			Status: api.StatusHealthy,
			Target: target,
		},
		{
			Time:   time.Date(2021, 1, 2, 3, 4, 6, 0, time.UTC),
			Status: api.StatusFailure,
			Target: target,
		},
		{
			Time:   time.Date(2021, 1, 2, 3, 4, 7, 0, time.UTC),
			Status: api.StatusHealthy,
			Target: target,
		},
	}

	t.Run("filter_by_status", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		q := query.ParseQuery("status=HEALTHY")
		fs := query.FilterScanner{Scanner: scanner, Query: q}

		var results []api.Record
		for fs.Scan() {
			results = append(results, fs.Record())
		}

		if len(results) != 2 {
			t.Errorf("expected 2 HEALTHY records, got %d", len(results))
		}
		for _, r := range results {
			if r.Status != api.StatusHealthy {
				t.Errorf("expected HEALTHY status, got %v", r.Status)
			}
		}
	})

	t.Run("filter_no_match", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		q := query.ParseQuery("status=UNKNOWN")
		fs := query.FilterScanner{Scanner: scanner, Query: q}

		var results []api.Record
		for fs.Scan() {
			results = append(results, fs.Record())
		}

		if len(results) != 0 {
			t.Errorf("expected 0 records, got %d", len(results))
		}
	})

	t.Run("filter_all_match", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		q := query.ParseQuery("target=*example.com*")
		fs := query.FilterScanner{Scanner: scanner, Query: q}

		var results []api.Record
		for fs.Scan() {
			results = append(results, fs.Record())
		}

		if len(results) != 3 {
			t.Errorf("expected 3 records, got %d", len(results))
		}
	})
}

func TestFilterScanner_Close(t *testing.T) {
	scanner := &mockLogScanner{logs: []api.Record{}}
	q := query.ParseQuery("")
	fs := query.FilterScanner{Scanner: scanner, Query: q}

	if err := fs.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !scanner.closed {
		t.Error("expected underlying scanner to be closed")
	}
}

func TestFilter(t *testing.T) {
	target, _ := api.ParseURL("https://example.com")

	logs := []api.Record{
		{
			Time:   time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC),
			Status: api.StatusHealthy,
			Target: target,
		},
		{
			Time:   time.Date(2021, 1, 2, 3, 4, 6, 0, time.UTC),
			Status: api.StatusFailure,
			Target: target,
		},
	}

	t.Run("returns_filter_scanner", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		filtered, since, until := query.Filter(scanner, "status=HEALTHY")

		var results []api.Record
		for filtered.Scan() {
			results = append(results, filtered.Record())
		}

		if len(results) != 1 {
			t.Errorf("expected 1 HEALTHY record, got %d", len(results))
		}

		// No time constraints in query
		if since != nil || until != nil {
			t.Error("expected nil time range for query without time constraints")
		}
	})

	t.Run("extracts_time_range_since", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		_, since, until := query.Filter(scanner, "time>=2021-01-02T00:00:00Z")

		if since == nil {
			t.Error("expected non-nil since time")
		} else {
			expected := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
			if !since.Equal(expected) {
				t.Errorf("expected since %v, got %v", expected, *since)
			}
		}

		if until != nil {
			t.Error("expected nil until time")
		}
	})

	t.Run("extracts_time_range_until", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		_, since, until := query.Filter(scanner, "time<=2021-01-03T00:00:00Z")

		if since != nil {
			t.Error("expected nil since time")
		}

		if until == nil {
			t.Error("expected non-nil until time")
		} else {
			// For <= operator, the until time is the end of the specified time unit
			expected := time.Date(2021, 1, 3, 0, 0, 0, 999999999, time.UTC)
			if !until.Equal(expected) {
				t.Errorf("expected until %v, got %v", expected, *until)
			}
		}
	})

	t.Run("extracts_time_range_both", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		_, since, until := query.Filter(scanner, "time>=2021-01-01T00:00:00Z time<=2021-01-31T00:00:00Z")

		if since == nil {
			t.Error("expected non-nil since time")
		}
		if until == nil {
			t.Error("expected non-nil until time")
		}
	})

	t.Run("empty_search_no_filter", func(t *testing.T) {
		scanner := &mockLogScanner{logs: logs}
		filtered, _, _ := query.Filter(scanner, "")

		var results []api.Record
		for filtered.Scan() {
			results = append(results, filtered.Record())
		}

		if len(results) != 2 {
			t.Errorf("expected 2 records (all), got %d", len(results))
		}
	})
}
