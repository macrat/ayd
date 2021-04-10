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
	tmpl := template.Must(template.New("status").Funcs(template.FuncMap{
		"each_runes": func(s string) []string {
			r := make([]string, len(s))
			for i, c := range []rune(s) {
				r[i] = string(c)
			}
			return r
		},
		"invert_incidents": func(xs []frozenIncident) []frozenIncident {
			rs := make([]frozenIncident, len(xs))
			for i, x := range xs {
				rs[len(xs)-i-1] = x
			}
			return rs
		},
	}).Parse(htmlTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		tmpl.Execute(w, freezeStatus(s))
	}
}
