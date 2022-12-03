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

func (r *fileScanner) seek(pos int64) {
	r.file.Seek(pos, os.SEEK_SET)
	r.reader = bufio.NewReader(r.file)
	r.pos = pos
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
	if s.Path() == "" {
		return newInMemoryScanner(s, since, until), nil
	}

	r, err := newFileScanner(s.Path(), since, until)
	if errors.Is(err, os.ErrNotExist) {
		return dummyScanner{}, nil
	} else {
		return r, err
	}
}
