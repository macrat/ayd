package endpoint

import (
	_ "embed"
	"encoding/json"
	"net/http"
	textTemplate "text/template"
)

//go:embed templates/status.html
var statusHTMLTemplate string

func StatusHTMLEndpoint(s Store) http.HandlerFunc {
	tmpl := loadHTMLTemplate(statusHTMLTemplate)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		handleError(s, "status.html", tmpl.Execute(newFlushWriter(w), s.MakeReport(20)))
	}
}

//go:embed templates/status.txt
var statusTextTemplate string

func StatusTextEndpoint(s Store) http.HandlerFunc {
	tmpl := textTemplate.Must(textTemplate.New("status.txt").Funcs(templateFuncs).Parse(statusTextTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		handleError(s, "status.txt", tmpl.Execute(newFlushWriter(w), s.MakeReport(40)))
	}
}

func StatusJSONEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(newFlushWriter(w))

		handleError(s, "status.json", enc.Encode(s.MakeReport(40)))
	}
}
