package endpoint

import (
	"bufio"
	_ "embed"
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

func NewLogGenerator(s Store, since, until time.Time) *LogGenerator {
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

func NewLogScanner(s Store, since, until time.Time) (LogScanner, error) {
	if s.Path() == "" {
		return NewLogGenerator(s, since, until), nil
	}
	return NewLogReader(s.Path(), since, until)
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

func getTimeQueries(s Store, scope string, r *http.Request, defaultPeriod time.Duration) (since, until time.Time, err error) {
	qs := r.URL.Query()

	var invalidQueries []string
	var errors []string

	until, err = getTimeQuery(qs, "until", time.Now())
	if err != nil {
		invalidQueries = append(invalidQueries, "until")
		errors = append(errors, err.Error())
	}

	since, err = getTimeQuery(qs, "since", until.Add(-defaultPeriod))
	if err != nil {
		invalidQueries = append(invalidQueries, "since")
		errors = append(errors, err.Error())
	}

	if len(invalidQueries) > 0 {
		handleError(s, scope, fmt.Errorf("%s", strings.Join(errors, "\n")))
		return since, until, fmt.Errorf("invalid query format: %s", strings.Join(invalidQueries, ", "))
	}

	return since, until, nil
}

func newLogScannerForEndpoint(s Store, scope string, r *http.Request) (scanner LogScanner, statusCode int, err error) {
	since, until, err := getTimeQueries(s, scope, r, 7*24*time.Hour)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	scanner, err = NewLogScanner(s, since, until)
	if err != nil {
		handleError(s, scope, fmt.Errorf("failed to open log: %w", err))
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	return scanner, http.StatusOK, nil
}

type LogFilter struct {
	Scanner LogScanner
	Targets []string
	Query   Query
}

type Query []string

func ParseQuery(query string) Query {
	var qs Query
	for _, q := range strings.Split(strings.ToLower(query), " ") {
		q = strings.TrimSpace(q)
		if q != "" {
			qs = append(qs, q)
		}
	}
	return qs
}

func (qs Query) Match(r api.Record) bool {
	target := strings.ToLower(r.Target.String())
	message := strings.ToLower(r.Message)

	for _, q := range qs {
		if !strings.Contains(target, q) && !strings.Contains(message, q) {
			return false
		}
	}
	return true
}

func setFilter(scanner LogScanner, r *http.Request) LogScanner {
	queries := r.URL.Query()

	targets := queries["target"]
	query := ParseQuery(queries.Get("query"))
	if len(targets) > 0 || len(query) > 0 {
		return LogFilter{scanner, targets, query}
	}

	return scanner
}

func (f LogFilter) Close() error {
	return f.Scanner.Close()
}

func (f LogFilter) filterByTarget(target string) bool {
	if len(f.Targets) == 0 {
		return true
	}
	for _, t := range f.Targets {
		if target == t {
			return true
		}
	}
	return false
}

func (f LogFilter) filterByQuery(r api.Record) bool {
	return f.Query.Match(f.Record())
}

func (f LogFilter) Scan() bool {
	for f.Scanner.Scan() {
		if f.filterByTarget(f.Record().Target.String()) && f.filterByQuery(f.Record()) {
			return true
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

func LogTSVEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/tab-separated-values; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		scanner, code, err := newLogScannerForEndpoint(s, "log.tsv", r)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		scanner = setFilter(scanner, r)

		for scanner.Scan() {
			_, err := w.Write(scanner.Bytes())
			if err != nil {
				handleError(s, "log.tsv", err)
				break
			}
		}
	}
}

func LogJsonEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(w)

		scanner, code, err := newLogScannerForEndpoint(s, "log.json", r)
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

		scanner = setFilter(scanner, r)

		records := struct {
			R []api.Record `json:"records"`
		}{
			[]api.Record{},
		}
		for scanner.Scan() {
			records.R = append(records.R, scanner.Record())
		}

		handleError(s, "log.json", enc.Encode(records))
	}
}

func LogCSVEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		scanner, code, err := newLogScannerForEndpoint(s, "log.csv", r)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		scanner = setFilter(scanner, r)

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

//go:embed templates/log.html
var logHTMLTemplate string

func LogHTMLEndpoint(s Store) http.HandlerFunc {
	tmpl := loadHTMLTemplate(logHTMLTemplate)

	return func(w http.ResponseWriter, r *http.Request) {
		since, until, err := getTimeQueries(s, "log.html", r, 1*time.Hour)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error() + "\n"))
			return
		}

		scanner, err := NewLogScanner(s, since, until)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		scanner = setFilter(scanner, r)

		var rs []api.Record
		for scanner.Scan() {
			rs = append(rs, scanner.Record())
		}

		var head []api.Record
		var tail []api.Record

		total := len(rs)
		if total > 20 {
			head = rs[:10]
			tail = rs[total-10:]
		} else {
			head = rs
		}
		count := len(head) + len(tail)

		query := strings.TrimSpace(r.URL.Query().Get("query"))

		rawQuery := url.Values{}
		rawQuery.Set("since", since.Format(time.RFC3339))
		rawQuery.Set("until", until.Format(time.RFC3339))
		rawQuery.Set("query", query)

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		handleError(s, "log.html", tmpl.Execute(w, logData{
			Since:    since,
			Until:    until,
			Query:    query,
			RawQuery: rawQuery.Encode(),
			Head:     head,
			Tail:     tail,
			Total:    total,
			Count:    count,
			Omitted:  total - count,
		}))
	}
}

type logData struct {
	Since    time.Time
	Until    time.Time
	Query    string
	RawQuery string
	Head     []api.Record
	Tail     []api.Record
	Total    int
	Count    int
	Omitted  int
}
