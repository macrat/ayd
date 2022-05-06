package endpoint

import (
	_ "embed"
	"net/http"
)

//go:embed templates/incidents.html
var incidentsHTMLTemplate string

func IncidentsHTMLEndpoint(s Store) http.HandlerFunc {
	tmpl := loadHTMLTemplate(incidentsHTMLTemplate)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		handleError(s, "incidents.html", tmpl.Execute(w, s.MakeReport(20)))
	}
}
