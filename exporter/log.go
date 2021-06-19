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

func newLogScannerForExporter(s *store.Store, w http.ResponseWriter, r *http.Request) (scanner LogScanner, ok bool) {
	qs := r.URL.Query()

	getTimeQuery := func(name string, default_ time.Time) (time.Time, error) {
		q := qs.Get(name)
		if q == "" {
			return default_, nil
		}

		t, err := time.Parse(time.RFC3339, q)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid `%s` format\n", name)
			return default_, fmt.Errorf("invalid %s format: %w", name, err)
		}
		return t, nil
	}

	since, err := getTimeQuery("since", time.Now().Add(-7*14*time.Hour))
	if err != nil {
		HandleError(s, "log.tsv", err)
		return nil, false
	}

	until, err := getTimeQuery("until", time.Now())
	if err != nil {
		HandleError(s, "log.tsv", err)
		return nil, false
	}

	scanner, err = NewLogScanner(s, since, until)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error\n"))
		HandleError(s, "log.tsv", fmt.Errorf("failed to open log: %w", err))
		return nil, false
	}

	return scanner, true
}

type LogFilter struct {
	Scanner LogScanner
	Targets []string
}

func (f LogFilter) Close() error {
	return f.Scanner.Close()
}

func (f LogFilter) Scan() bool {
	for f.Scanner.Scan() {
		for _, t := range f.Targets {
			if f.Record().Target.String() == t {
				return true
			}
		}
	}
	return false
}

func (f LogFilter) Bytes() []byte {
	return f.Scanner.Bytes()
}

func (f LogFilter) Record() api.Record {
	return f.Scanner.Record()
}

func LogTSVExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/tab-separated-values; charset=UTF-8")

		scanner, ok := newLogScannerForExporter(s, w, r)
		if !ok {
			return
		}
		defer scanner.Close()

		if targets, ok := r.URL.Query()["target"]; ok {
			scanner = LogFilter{scanner, targets}
		}

		for scanner.Scan() {
			w.Write(scanner.Bytes())
		}
	}
}
