package endpoint

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/macrat/ayd/internal/store"
)

// TargetsTextEndpoint replies target list in text.
func TargetsTextEndpoint(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		for _, t := range s.Targets() {
			fmt.Fprintln(w, t)
		}
	}
}

// TargetsJSONEndpoint replies target list in json format.
func TargetsJSONEndpoint(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)

		HandleError(s, "targets.json", enc.Encode(s.Targets()))
	}
}
