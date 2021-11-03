package endpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// TargetsTextEndpoint replies target list in text.
func TargetsTextEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		for _, t := range s.Targets() {
			fmt.Fprintln(w, t)
		}
	}
}

// TargetsJSONEndpoint replies target list in json format.
func TargetsJSONEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)

		HandleError(s, "targets.json", enc.Encode(s.Targets()))
	}
}
