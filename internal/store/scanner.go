package store

import (
	"bufio"
	"errors"
	"os"
	"sort"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type fileScanner struct {
	file   *os.File
	reader *bufio.Reader
	since  time.Time
	until  time.Time
	rec    api.Record
	pos    int64
}

// newFileScanner creates a new [fileScanner] from file path, with period specification.
func newFileScanner(path string, since, until time.Time) (*fileScanner, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &fileScanner{
		file:   f,
		reader: bufio.NewReader(f),
		since:  since,
		until:  until,
	}, nil
}

func (r *fileScanner) Close() error {
	return r.file.Close()
}

func (r *fileScanner) Scan() bool {
	for {
		b, err := r.reader.ReadBytes('\n')
		if err != nil {
			return false
		}
		r.pos += int64(len(b))

		var rec api.Record
		err = rec.UnmarshalJSON(b)
		if err == nil && !rec.Time.Before(r.since) && r.until.After(rec.Time) {
			r.rec = rec
			return true
		}
	}
}

func (r *fileScanner) Record() api.Record {
	return r.rec
}

type fileScannerSet struct {
	scanners []*fileScanner
	scanned  bool
	earliest int
}

func newFileScannerSet(pathes []string, since, until time.Time) (*fileScannerSet, error) {
	min := time.Unix(1<<63-1, 0)

	var ss fileScannerSet
	for i, p := range pathes {
		s, err := newFileScanner(p, since, until)
		if err != nil {
			ss.Close()
			return nil, err
		}
		if !s.Scan() {
			ss.Close()
			continue
		}
		if s.Record().Time.Before(min) {
			ss.earliest = i
			min = s.Record().Time
		}
		ss.scanners = append(ss.scanners, s)
	}
	return &ss, nil
}

func (r *fileScannerSet) Close() error {
	var err error
	for _, s := range r.scanners {
		if e := s.Close(); e != nil {
			err = e
		}
	}
	return err
}

func (r *fileScannerSet) updateEarliest() {
	max := time.Unix(1<<63-1, 0)
	for i, s := range r.scanners {
		if s.Record().Time.Before(max) {
			r.earliest = i
			max = s.Record().Time
		}
	}
}

func (r *fileScannerSet) Scan() bool {
	if !r.scanned {
		r.scanned = true
		return len(r.scanners) > 0
	}

	for len(r.scanners) > 0 {
		if r.scanners[r.earliest].Scan() {
			r.updateEarliest()
			return true
		} else {
			r.scanners[r.earliest].Close()
			r.scanners = append(r.scanners[:r.earliest], r.scanners[r.earliest+1:]...)
			r.updateEarliest()
		}
	}

	return false
}

func (r *fileScannerSet) Record() api.Record {
	if len(r.scanners) < 0 {
		panic("This is a bug if you see this message.")
	}
	return r.scanners[r.earliest].Record()
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

type dummyScanner struct{}

func (r dummyScanner) Close() error {
	return nil
}

func (r dummyScanner) Scan() bool {
	return false
}

func (r dummyScanner) Record() api.Record {
	// This method never be called.
	panic("This is a bug if you see this message.")
}

func (s *Store) OpenLog(since, until time.Time) (api.LogScanner, error) {
	if s.path.IsEmpty() {
		return newInMemoryScanner(s, since, until), nil
	}

	r, err := newFileScannerSet(s.path.ListBetween(since, until), since, until)
	if errors.Is(err, os.ErrNotExist) {
		return dummyScanner{}, nil
	} else {
		return r, err
	}
}
