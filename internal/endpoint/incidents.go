package endpoint

import (
	_ "embed"
	"net/http"
	"os"
	"text/template"

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

		report := s.MakeReport(0)

		handleError(s, "incidents.rss", tmpl.Execute(w, incidentsInfo{
			ExternalURL: os.Getenv("AYD_URL"),
			Report:      report,
		}))
	}
}

type incidentsInfo struct {
	ExternalURL string
	Report      api.Report
}
