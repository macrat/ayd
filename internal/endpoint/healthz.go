package endpoint

import (
	"fmt"
	"net/http"
)

// HealthzEndpoint is the http.HandlerFunc for /healthz page.
func HealthzEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		healthy, messages := s.Errors()

		if healthy {
			fmt.Fprintln(w, "HEALTHY")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "FAILURE")
		}

		for _, msg := range messages {
			fmt.Fprintln(w, msg)
		}
	}
}
