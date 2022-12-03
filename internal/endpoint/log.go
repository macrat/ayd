package endpoint

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func getTimeQuery(queries url.Values, name string, default_ time.Time) (time.Time, error) {
	q := queries.Get(name)
	if q == "" {
		return default_, nil
	}

	t, err := api.ParseTime(q)
	if err != nil {
		return default_, fmt.Errorf("invalid %s format: %q", name, q)
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

func newLogScanner(s Store, scope string, r *http.Request) (scanner api.LogScanner, statusCode int, err error) {
	since, until, err := getTimeQueries(s, scope, r, 7*24*time.Hour)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	scanner, err = s.OpenLog(since, until)
	if err != nil {
		handleError(s, scope, fmt.Errorf("failed to open log: %w", err))
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	return scanner, http.StatusOK, nil
}

type LogFilter struct {
	Scanner api.LogScanner
	Targets []string
	Query   Query
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

func setFilter(scanner api.LogScanner, r *http.Request) api.LogScanner {
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

func (f LogFilter) Record() api.Record {
	return f.Scanner.Record()
}

func LogJsonEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(w)

		scanner, code, err := newLogScanner(s, "log.json", r)
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

		scanner, code, err := newLogScanner(s, "log.csv", r)
		if err != nil {
			w.WriteHeader(code)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
		defer scanner.Close()

		scanner = setFilter(scanner, r)

		c := csv.NewWriter(w)
		c.Write([]string{"timestamp", "status", "latency", "target", "message", "extra"})

		for scanner.Scan() {
			r := scanner.Record()

			var extra []byte
			if len(r.Extra) > 0 {
				extra, _ = json.Marshal(r.Extra) // Ignore error because it use empty string if failed to convert.
			}

			c.Write([]string{
				r.Time.Format(time.RFC3339),
				r.Status.String(),
				strconv.FormatFloat(float64(r.Latency.Microseconds())/1000, 'f', 3, 64),
				r.Target.String(),
				r.Message,
				string(extra),
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

		scanner, err := s.OpenLog(since, until)
		if err != nil {
			handleError(s, "log.html", fmt.Errorf("failed to open log: %w", err))
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
