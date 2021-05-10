package exporter

import (
	_ "embed"
	"net/http"
	"text/template"

	"github.com/macrat/ayd/store"
)

//go:embed templates/status.txt
var textTemplate string

func TextExporter(s *store.Store) http.HandlerFunc {
	tmpl := template.Must(template.New("status.txt").Funcs(templateFuncs).Parse(textTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		HandleError(s, "status.txt", tmpl.Execute(w, s.Freeze()))
	}
}
