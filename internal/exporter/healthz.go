package exporter

import (
	"fmt"
	"net/http"
)

type ErrorsGetter interface {
	Errors() (healthy bool, messages []string)
}

// HealthzExporter is the http.HandlerFunc for /healthz page.
// It receives ErrorsGetter interface instead of *store.Store because for make easier to test.
func HealthzExporter(s ErrorsGetter) http.HandlerFunc {
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
