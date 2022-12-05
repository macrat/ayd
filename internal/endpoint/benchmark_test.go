package endpoint_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
)

func Benchmark_endpoints(b *testing.B) {
	benchmarks := []struct {
		Path     string
		Endpoint func(endpoint.Store) http.HandlerFunc
	}{
		{"/status.html", endpoint.StatusHTMLEndpoint},
		{"/status.txt", endpoint.StatusTextEndpoint},
		{"/status.json", endpoint.StatusJSONEndpoint},
		{"/incidents.html", endpoint.IncidentsHTMLEndpoint},
		{"/incidents.rss", endpoint.IncidentsRSSEndpoint},
		{"/incidents.csv", endpoint.IncidentsCSVEndpoint},
		{"/log.html", endpoint.LogHTMLEndpoint},
		{"/log.csv", endpoint.LogCSVEndpoint},
		{"/log.ltsv", endpoint.LogLTSVEndpoint},
		{"/log.json", endpoint.LogJsonEndpoint},
		{"/metrics", endpoint.MetricsEndpoint},
		{"/healthz", func(s endpoint.Store) http.HandlerFunc {
			return endpoint.HealthzEndpoint(s)
		}},
	}

	for _, tt := range benchmarks {
		b.Run(tt.Path, func(b *testing.B) {
			s := testutil.NewStoreWithLog(b)
			defer s.Close()

			h := tt.Endpoint(s)

			r := httptest.NewRequest("GET", tt.Path, nil)

			var buf bytes.Buffer

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				w.Body = &buf
				h(w, r)
				buf.Reset()
			}
		})
	}
}
