package endpoint

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"text/template"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

//go:embed templates/incidents.html
var incidentsHTMLTemplate string

func IncidentsHTMLEndpoint(s Store) http.HandlerFunc {
	tmpl := loadHTMLTemplate(incidentsHTMLTemplate)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		handleError(s, "incidents.html", tmpl.Execute(w, s.MakeReport(0)))
	}
}

//go:embed templates/incidents.rss
var incidentsRSSTemplate string

func IncidentsRSSEndpoint(s Store) http.HandlerFunc {
	tmpl := template.Must(template.New("incidents.rss").Funcs(templateFuncs).Parse(incidentsRSSTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")

		handleError(s, "incidents.rss", tmpl.Execute(w, newIncidentsInfo(s)))
	}
}

func IncidentsCSVEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		c := csv.NewWriter(w)
		c.Write([]string{"caused_at", "resolved_at", "status", "target", "message"})

		rs := newIncidentsInfo(s).Incidents

		for _, r := range rs {
			resolved := ""
			if !r.ResolvedAt.IsZero() {
				resolved = r.ResolvedAt.Format(time.RFC3339)
			}

			c.Write([]string{
				r.CausedAt.Format(time.RFC3339),
				resolved,
				r.Status.String(),
				r.Target.Redacted(),
				r.Message,
			})
		}

		c.Flush()
	}
}

func IncidentsJSONEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(w)

		handleError(s, "log.json", enc.Encode(newIncidentsInfo(s)))
	}
}

type incidentsInfo struct {
	ExternalURL string         `json:"-"`
	Incidents   []api.Incident `json:"incidents"`
	ReportedAt  time.Time      `json:"reported_at"`
}

func newIncidentsInfo(s Store) incidentsInfo {
	report := s.MakeReport(0)

	rs := append(report.IncidentHistory, report.CurrentIncidents...)
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].CausedAt.Equal(rs[j].CausedAt) {
			return rs[i].Target.Redacted() < rs[j].Target.Redacted()
		}
		return rs[i].CausedAt.Before(rs[j].CausedAt)
	})

	return incidentsInfo{
		ExternalURL: os.Getenv("AYD_URL"),
		Incidents:   rs,
		ReportedAt:  report.ReportedAt,
	}
}
