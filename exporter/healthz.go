package exporter

import (
	"fmt"
	"net/http"

	"github.com/macrat/ayd/store"
)

func HealthzExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		if err := s.Err(); err == nil {
			fmt.Fprintln(w, "OK")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "FAILURE")
			fmt.Fprintln(w, err)
		}
	}
}
