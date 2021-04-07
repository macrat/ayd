package exporter

import (
	"net/http"

	"github.com/macrat/ayd/store"
)

func New(s *store.Store) http.Handler {
	m := http.NewServeMux()

	m.HandleFunc("/status.txt", TextExporter(s))
	m.HandleFunc("/status.json", JSONExporter(s))

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/status.txt", http.StatusFound)
	})

	return m
}
