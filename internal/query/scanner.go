package query

import (
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

// FilterScanner wraps a LogScanner and filters records by a query.
type FilterScanner struct {
	Scanner api.LogScanner
	Query   Query
}

// Close closes the underlying scanner.
func (f FilterScanner) Close() error {
	return f.Scanner.Close()
}

// Scan advances to the next matching record.
func (f FilterScanner) Scan() bool {
	for f.Scanner.Scan() {
		if f.Query.Match(f.Record()) {
			return true
		}
	}
	return false
}

// Record returns the current record.
func (f FilterScanner) Record() api.Record {
	return f.Scanner.Record()
}

// Filter wraps a LogScanner with a search query filter.
// It returns the filtered scanner and the time range extracted from the query.
func Filter(scanner api.LogScanner, search string) (api.LogScanner, *time.Time, *time.Time) {
	q := ParseQuery(search)
	st, en := q.TimeRange()
	return FilterScanner{Scanner: scanner, Query: q}, st, en
}
