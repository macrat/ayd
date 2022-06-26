package store

import (
	"os"
	"sort"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func newFileScanner(path string, since, until time.Time) (api.LogScanner, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return api.NewLogScannerWithPeriod(f, since, until), nil
}

type inMemoryScanner struct {
	records []api.Record
	index   int
}

func newInMemoryScanner(s *Store, since, until time.Time) *inMemoryScanner {
	r := &inMemoryScanner{index: -1}
	for _, xs := range s.ProbeHistory() {
		for _, x := range xs.Records {
			if !x.Time.Before(since) && x.Time.Before(until) {
				r.records = append(r.records, x)
			}
		}
	}
	sort.Sort(r)
	return r
}

func (r inMemoryScanner) Len() int {
	return len(r.records)
}

func (r inMemoryScanner) Less(i, j int) bool {
	if !r.records[i].Time.Equal(r.records[j].Time) {
		return r.records[i].Time.Before(r.records[j].Time)
	}
	return r.records[i].Target.String() < r.records[j].Target.String()
}

func (r inMemoryScanner) Swap(i, j int) {
	r.records[i], r.records[j] = r.records[j], r.records[i]
}

func (r *inMemoryScanner) Close() error {
	return nil
}

func (r *inMemoryScanner) Scan() bool {
	if r.index+1 >= len(r.records) {
		return false
	}
	r.index++
	return true
}

func (r *inMemoryScanner) Record() api.Record {
	return r.records[r.index]
}

func (s *Store) OpenLog(since, until time.Time) (api.LogScanner, error) {
	if s.Path() == "" {
		return newInMemoryScanner(s, since, until), nil
	}
	return newFileScanner(s.Path(), since, until)
}
