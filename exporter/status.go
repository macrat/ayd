package exporter

import (
	_ "embed"
	"encoding/json"
	htmlTemplate "html/template"
	"net/http"
	textTemplate "text/template"

	"github.com/macrat/ayd/store"
)

//go:embed templates/status.html
var statusHTMLTemplate string

func StatusHTMLExporter(s *store.Store) http.HandlerFunc {
	tmpl := htmlTemplate.Must(htmlTemplate.New("status.html").Funcs(templateFuncs).Parse(statusHTMLTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		HandleError(s, "status.html", tmpl.Execute(w, s.Freeze()))
	}
}

//go:embed templates/status.txt
var statusTextTemplate string

func StatusTextExporter(s *store.Store) http.HandlerFunc {
	tmpl := textTemplate.Must(textTemplate.New("status.txt").Funcs(templateFuncs).Parse(statusTextTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		HandleError(s, "status.txt", tmpl.Execute(w, s.Freeze()))
	}
}

func StatusJSONExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Controll-Allow-Origin", "*")
		w.Header().Set("Access-Controll-Allow-Methods", "GET")

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		HandleError(s, "status.json", enc.Encode(s.Freeze()))
	}
}
