package exporter

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
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

func getTimeQuery(queries url.Values, name string, default_ time.Time) (time.Time, error) {
	q := queries.Get(name)
	if q == "" {
		return default_, nil
	}

	t, err := time.Parse(time.RFC3339, q)
	if err != nil {
		return default_, fmt.Errorf("invalid %s format: %w", name, err)
	}
	return t, nil
}

func newLogScannerForExporter(s *store.Store, r *http.Request) (scanner LogScanner, statusCode int, err error) {
	qs := r.URL.Query()

	var invalidQueries []string
	var errors []string

	since, err := getTimeQuery(qs, "since", time.Now().Add(-7*14*time.Hour))
	if err != nil {
		invalidQueries = append(invalidQueries, "since")
		errors = append(errors, err.Error())
	}

	until, err := getTimeQuery(qs, "until", time.Now())
	if err != nil {
		invalidQueries = append(invalidQueries, "until")
		errors = append(errors, err.Error())
	}

	if len(invalidQueries) > 0 {
		HandleError(s, "log.tsv", fmt.Errorf("%s", strings.Join(errors, "\n")))
		return nil, http.StatusBadRequest, fmt.Errorf("invalid query format: %s", strings.Join(invalidQueries, ", "))
	}

	scanner, err = NewLogScanner(s, since, until)
	if err != nil {
		HandleError(s, "log.tsv", fmt.Errorf("failed to open log: %w", err))
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	return scanner, http.StatusOK, nil
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		scanner, code, err := newLogScannerForExporter(s, r)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		if targets, ok := r.URL.Query()["target"]; ok {
			scanner = LogFilter{scanner, targets}
		}

		for scanner.Scan() {
			_, err := w.Write(scanner.Bytes())
			if err != nil {
				HandleError(s, "log.tsv", err)
				break
			}
		}
	}
}

func LogJsonExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(w)

		scanner, code, err := newLogScannerForExporter(s, r)
		if err != nil {
			msg := struct {
				E string `json:"error"`
			}{
				err.Error(),
			}

			w.WriteHeader(code)
			enc.Encode(msg)
			return
		}
		defer scanner.Close()

		if targets, ok := r.URL.Query()["target"]; ok {
			scanner = LogFilter{scanner, targets}
		}

		records := struct {
			R []api.Record `json:"records"`
		}{}
		for scanner.Scan() {
			records.R = append(records.R, scanner.Record())
		}

		HandleError(s, "log.json", enc.Encode(records))
	}
}

func LogCSVExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		scanner, code, err := newLogScannerForExporter(s, r)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		if targets, ok := r.URL.Query()["target"]; ok {
			scanner = LogFilter{scanner, targets}
		}

		c := csv.NewWriter(w)
		c.Write([]string{"timestamp", "status", "latency", "target", "message"})

		for scanner.Scan() {
			r := scanner.Record()
			c.Write([]string{
				r.CheckedAt.Format(time.RFC3339),
				r.Status.String(),
				strconv.FormatFloat(float64(r.Latency.Microseconds())/1000, 'f', -1, 64),
				r.Target.Redacted(),
				r.Message,
			})
		}

		c.Flush()
	}
}
