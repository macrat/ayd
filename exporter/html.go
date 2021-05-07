package exporter

import (
	_ "embed"
	"html/template"
	"net/http"

	"github.com/macrat/ayd/store"
)

//go:embed templates/status.html
var htmlTemplate string

func HTMLExporter(s *store.Store) http.HandlerFunc {
	tmpl := template.Must(template.New("status.html").Funcs(templateFuncs).Parse(htmlTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		tmpl.Execute(w, s.Freeze())
	}
}
