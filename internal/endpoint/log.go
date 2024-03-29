package endpoint

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/macrat/ayd/internal/logconv"
	api "github.com/macrat/ayd/lib-ayd"
)

func getTimeQuery(queries url.Values, name string, default_ time.Time) (time.Time, error) {
	q := queries.Get(name)
	if q == "" {
		return default_, nil
	}

	if n, err := strconv.ParseInt(q, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}

	t, err := api.ParseTime(q)
	if err != nil {
		return default_, fmt.Errorf("invalid %s format: %q", name, q)
	}
	return t, nil
}

type logOptions struct {
	Since, Until  time.Time
	Limit, Offset uint64
	Targets       []string
	Query         Query
}

func newLogOptionsByRequest(s Store, scope string, r *http.Request, defaultPeriod time.Duration) (opts logOptions, err error) {
	var invalidQueries []string
	var errors []string

	qs := r.URL.Query()

	opts.Until, err = getTimeQuery(qs, "until", time.Now())
	if err != nil {
		invalidQueries = append(invalidQueries, "until")
		errors = append(errors, err.Error())
	}

	opts.Since, err = getTimeQuery(qs, "since", opts.Until.Add(-defaultPeriod))
	if err != nil {
		invalidQueries = append(invalidQueries, "since")
		errors = append(errors, err.Error())
	}

	if l := qs.Get("limit"); l != "" {
		opts.Limit, err = strconv.ParseUint(l, 10, 64)
		if err != nil {
			invalidQueries = append(invalidQueries, "limit")
			errors = append(errors, err.Error())
		}
	}

	if o := qs.Get("offset"); o != "" {
		opts.Offset, err = strconv.ParseUint(o, 10, 64)
		if err != nil {
			invalidQueries = append(invalidQueries, "offset")
			errors = append(errors, err.Error())
		}
	}

	opts.Targets = qs["target"]

	if q := ParseQuery(qs.Get("query")); len(qs) > 0 {
		opts.Query = q
	}

	if len(invalidQueries) > 0 {
		handleError(s, scope, fmt.Errorf("%s", strings.Join(errors, "\n")))
		return opts, fmt.Errorf("invalid query format: %s", strings.Join(invalidQueries, ", "))
	}

	return opts, nil
}

type PagingScanner struct {
	Scanner api.LogScanner
	Offset  uint64
	Limit   uint64
	count   uint64
}

func (s *PagingScanner) Scan() bool {
	for s.count < s.Offset {
		if !s.Scanner.Scan() {
			return false
		}
		s.count++
	}
	if s.Limit != 0 && s.Limit <= s.count-s.Offset {
		return false
	}
	ok := s.Scanner.Scan()
	if ok {
		s.count++
	}
	return ok
}

// ScanTotal scans all logs and return number of records.
// Don't call this before get records you need, because this method consumes all logs.
func (s *PagingScanner) ScanTotal() uint64 {
	for s.Scanner.Scan() {
		s.count++
	}
	return s.count
}

func (s *PagingScanner) Record() api.Record {
	return s.Scanner.Record()
}

func (s *PagingScanner) Close() error {
	return s.Scanner.Close()
}

type FilterScanner struct {
	Scanner api.LogScanner
	Targets []string
	Query   Query
}

func (f FilterScanner) Close() error {
	return f.Scanner.Close()
}

func (f FilterScanner) filterByTarget(target string) bool {
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

func (f FilterScanner) filterByQuery(r api.Record) bool {
	return f.Query.Match(f.Record())
}

func (f FilterScanner) Scan() bool {
	for f.Scanner.Scan() {
		if f.filterByTarget(f.Record().Target.String()) && f.filterByQuery(f.Record()) {
			return true
		}
	}
	return false
}

func (f FilterScanner) Record() api.Record {
	return f.Scanner.Record()
}

type keyword interface {
	Match(status string, latency time.Duration, target, message string) bool
}

type strKeyword string

func (k strKeyword) Match(status string, latency time.Duration, target, message string) bool {
	q := string(k)
	return status == q || strings.Contains(target, q) || strings.Contains(message, q)
}

type strNotKeyword string

func (k strNotKeyword) Match(status string, latency time.Duration, target, message string) bool {
	return !strKeyword(k).Match(status, latency, target, message)
}

type durKeyword struct {
	operator string
	duration time.Duration
}

func parseDurKeyword(s string) (result durKeyword, ok bool) {
	for _, operator := range []string{"<=", "<", ">=", ">", "!=", "="} {
		if strings.HasPrefix(s, operator) {
			result.operator = operator
			var err error
			result.duration, err = time.ParseDuration(s[len(operator):])
			return result, err == nil
		}
	}
	return result, false
}

func (k durKeyword) Match(status string, latency time.Duration, target, message string) bool {
	switch k.operator {
	case "<":
		return latency < k.duration
	case "<=":
		return latency <= k.duration
	case ">":
		return latency > k.duration
	case ">=":
		return latency >= k.duration
	case "!=":
		return latency != k.duration
	case "=":
		return latency == k.duration
	default:
		return false
	}
}

type Query []keyword

func ParseQuery(query string) Query {
	var qs Query
	for _, q := range strings.Split(strings.ToLower(query), " ") {
		q = strings.TrimSpace(q)
		if q != "" {
			if dur, ok := parseDurKeyword(q); ok {
				qs = append(qs, dur)
			} else if len(q) > 2 && q[0] == '-' {
				qs = append(qs, strNotKeyword(q[1:]))
			} else {
				qs = append(qs, strKeyword(q))
			}
		}
	}
	return qs
}

func (qs Query) Match(r api.Record) bool {
	status := strings.ToLower(r.Status.String())
	target := strings.ToLower(r.Target.String())
	message := strings.ToLower(r.ReadableMessage())

	for _, q := range qs {
		if !q.Match(status, r.Latency, target, message) {
			return false
		}
	}
	return true
}

type ContextScanner struct {
	ctx     context.Context
	scanner api.LogScanner
}

func NewContextScanner(ctx context.Context, s api.LogScanner) ContextScanner {
	return ContextScanner{ctx, s}
}

func (cs ContextScanner) Scan() bool {
	select {
	case <-cs.ctx.Done():
		return false
	default:
		return cs.scanner.Scan()
	}
}

func (cs ContextScanner) Record() api.Record {
	return cs.scanner.Record()
}

func (cs ContextScanner) Close() error {
	return cs.scanner.Close()
}

func newLogScanner(s Store, scope string, r *http.Request, defaultPeriod time.Duration) (scanner *PagingScanner, statusCode int, err error) {
	opts, err := newLogOptionsByRequest(s, scope, r, defaultPeriod)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	return newLogScannerByOpts(s, scope, r, opts)
}

func newLogScannerByOpts(s Store, scope string, r *http.Request, opts logOptions) (scanner *PagingScanner, statusCode int, err error) {
	rawScanner, err := s.OpenLog(opts.Since, opts.Until)
	if err != nil {
		handleError(s, scope, fmt.Errorf("failed to open log: %w", err))
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	rawScanner = NewContextScanner(r.Context(), rawScanner)

	rawScanner = FilterScanner{
		Scanner: rawScanner,
		Targets: opts.Targets,
		Query:   opts.Query,
	}

	scanner = &PagingScanner{
		Scanner: rawScanner,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
	}

	return scanner, http.StatusOK, nil
}

func LogJsonEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(newFlushWriter(w))

		opts, err := newLogOptionsByRequest(s, "log.json", r, 7*24*time.Hour)
		if err != nil {
			msg := struct {
				E string `json:"error"`
			}{
				err.Error(),
			}

			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(msg)
			return
		}

		scanner, code, err := newLogScannerByOpts(s, "log.json", r, opts)
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

		records := struct {
			R []api.Record `json:"records"`
			T uint64       `json:"total"`
			P string       `json:"prev,omitempty"`
			N string       `json:"next,omitempty"`
		}{
			R: []api.Record{},
		}
		for scanner.Scan() {
			records.R = append(records.R, scanner.Record())
		}
		records.T = scanner.ScanTotal()

		u := r.URL
		if next := opts.Offset + uint64(len(records.R)); next < records.T {
			q := u.Query()
			q.Set("offset", strconv.FormatUint(next, 10))
			u.RawQuery = q.Encode()
			records.N = u.String()
		}
		if opts.Offset > 0 {
			var prev uint64
			if opts.Offset > opts.Limit {
				prev = opts.Offset - opts.Limit
			}
			q := u.Query()
			q.Set("offset", strconv.FormatUint(prev, 10))
			u.RawQuery = q.Encode()
			records.P = u.String()
		}

		handleError(s, "log.json", enc.EncodeContext(r.Context(), records))
	}
}

func LogCSVEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		scanner, code, err := newLogScanner(s, "log.csv", r, 7*24*time.Hour)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		err = logconv.ToCSV(newFlushWriter(w), scanner)
		handleError(s, "log.csv", err)
	}
}

func LogLTSVEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		scanner, code, err := newLogScanner(s, "log.ltsv", r, 7*24*time.Hour)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		err = logconv.ToLTSV(newFlushWriter(w), scanner)
		handleError(s, "log.ltsv", err)
	}
}

func LogXlsxEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

		scanner, code, err := newLogScanner(s, "log.xlsx", r, 7*24*time.Hour)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		err = logconv.ToXlsx(newFlushWriter(w), scanner, time.Now())
		handleError(s, "log.xlsx", err)
	}
}

//go:embed templates/log.html
var logHTMLTemplate string

func LogHTMLEndpoint(s Store) http.HandlerFunc {
	tmpl := loadHTMLTemplate(logHTMLTemplate)

	return func(w http.ResponseWriter, r *http.Request) {
		opts, err := newLogOptionsByRequest(s, "log.html", r, time.Hour)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error() + "\n"))
			return
		}

		if opts.Limit == 0 {
			opts.Limit = 25
		}

		scanner, code, err := newLogScannerByOpts(s, "log.html", r, opts)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}

		var rs []api.Record
		for scanner.Scan() {
			rs = append(rs, scanner.Record())
		}

		total := scanner.ScanTotal()

		query := strings.TrimSpace(r.URL.Query().Get("query"))

		rawQuery := url.Values{}
		rawQuery.Set("since", opts.Since.Format(time.RFC3339))
		rawQuery.Set("until", opts.Until.Format(time.RFC3339))
		rawQuery.Set("query", query)

		var prev uint64
		if opts.Offset > opts.Limit {
			prev = opts.Offset - opts.Limit
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		handleError(s, "log.html", tmpl.Execute(newFlushWriter(w), logData{
			Since:    opts.Since,
			Until:    opts.Until,
			Query:    query,
			RawQuery: rawQuery.Encode(),
			Records:  rs,
			Total:    total,
			From:     opts.Offset + 1,
			To:       opts.Offset + uint64(len(rs)),
			Prev:     prev,
			Next:     opts.Offset + uint64(len(rs)),
			Limit:    opts.Limit,
		}))
	}
}

type logData struct {
	Since    time.Time
	Until    time.Time
	Query    string
	RawQuery string
	Records  []api.Record
	Total    uint64
	From     uint64
	To       uint64
	Prev     uint64
	Next     uint64
	Limit    uint64
}
