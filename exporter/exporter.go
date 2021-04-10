package exporter

import (
	_ "embed"
	"net/http"

	"github.com/macrat/ayd/store"
)

//go:embed static/favicon.ico
var faviconIco []byte

//go:embed static/favicon.svg
var faviconSvg []byte

func New(s *store.Store) http.Handler {
	m := http.NewServeMux()

	m.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconIco)
	})
	m.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconSvg)
	})

	m.HandleFunc("/status.txt", TextExporter(s))
	m.HandleFunc("/status.json", JSONExporter(s))

	m.Handle("/", http.RedirectHandler("/status.txt", http.StatusFound))

	return m
}
