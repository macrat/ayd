package exporter

import (
	"encoding/json"
	"net/http"

	"github.com/macrat/ayd/store"
)

func JSONExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Controll-Allow-Origin", "*")
		w.Header().Set("Access-Controll-Allow-Methods", "GET")

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		enc.Encode(s.Freeze())
	}
}
