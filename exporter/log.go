package exporter

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/store"
)

type LogReader struct {
	file    io.ReadCloser
	scanner *bufio.Scanner
	since   time.Time
	until   time.Time
	rec     api.Record
}

func NewLogReaderFromReader(f io.ReadCloser, since, until time.Time) *LogReader {
	return &LogReader{
		file:    f,
		scanner: bufio.NewScanner(f),
		since:   since,
		until:   until,
	}
}

func NewLogReader(path string, since, until time.Time) (*LogReader, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return NewLogReaderFromReader(f, since, until), nil
}

func (r *LogReader) Close() error {
	return r.file.Close()
}

func (r *LogReader) Scan() bool {
	for r.scanner.Scan() {
		rec, err := api.ParseRecord(r.scanner.Text())
		if err != nil || rec.CheckedAt.Before(r.since) {
			continue
		}
		if !r.until.After(rec.CheckedAt) {
			return false
		}
		r.rec = rec
		return true
	}
	return false
}

func (r *LogReader) Bytes() []byte {
	return append(r.scanner.Bytes(), byte('\n'))
}

func (r *LogReader) Record() api.Record {
	return r.rec
}

type LogGenerator struct {
	records []api.Record
	index   int
}

func NewLogGenerator(s *store.Store, since, until time.Time) *LogGenerator {
	g := &LogGenerator{index: -1}
	for _, xs := range s.ProbeHistory() {
		for _, x := range xs.Records {
			if !x.CheckedAt.Before(since) && x.CheckedAt.Before(until) {
				g.records = append(g.records, x)
			}
		}
	}
	sort.Sort(g)
	return g
}

func (g LogGenerator) Len() int {
	return len(g.records)
}

func (g LogGenerator) Less(i, j int) bool {
	return g.records[i].CheckedAt.Before(g.records[j].CheckedAt)
}

func (g LogGenerator) Swap(i, j int) {
	g.records[i], g.records[j] = g.records[j], g.records[i]
}

func (g *LogGenerator) Close() error {
	return nil
}

func (g *LogGenerator) Scan() bool {
	if g.index+1 >= len(g.records) {
		return false
	}
	g.index++
	return true
}

func (g *LogGenerator) Bytes() []byte {
	return []byte(g.records[g.index].String() + "\n")
}

func (g *LogGenerator) Record() api.Record {
	return g.records[g.index]
}

type LogScanner interface {
	Close() error
	Scan() bool
	Bytes() []byte
	Record() api.Record
}

func NewLogScanner(s *store.Store, since, until time.Time) (LogScanner, error) {
	if s.Path == "" {
		return NewLogGenerator(s, since, until), nil
	}
	return NewLogReader(s.Path, since, until)
}

func LogTSVExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/tab-separated-values; charset=UTF-8")

		until := time.Now()
		since := until.Add(-7 * 14 * time.Hour)

		var err error

		qs := r.URL.Query()
		if q := qs.Get("since"); q != "" {
			since, err = time.Parse(time.RFC3339, q)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("invalid `since` format\n"))
				HandleError(s, "log.tsv", fmt.Errorf("invalid since format: %w", err))
				return
			}
		}
		if q := qs.Get("until"); q != "" {
			until, err = time.Parse(time.RFC3339, q)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("invalid `until` format\n"))
				HandleError(s, "log.tsv", fmt.Errorf("invalid until format: %w", err))
				return
			}
		}

		reader, err := NewLogScanner(s, since, until)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error\n"))
			HandleError(s, "log.tsv", fmt.Errorf("failed to open log: %w", err))
			return
		}

		for reader.Scan() {
			w.Write(reader.Bytes())
		}
	}
}
