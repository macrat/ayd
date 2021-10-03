package exporter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/internal/exporter"
	"github.com/macrat/ayd/internal/store"
	"github.com/macrat/ayd/internal/testutil"
)

func Benchmark_exporters(b *testing.B) {
	benchmarks := []struct {
		Path     string
		Exporter func(*store.Store) http.HandlerFunc
	}{
		{"/status.html", exporter.StatusHTMLExporter},
		{"/status.txt", exporter.StatusTextExporter},
		{"/status.json", exporter.StatusJSONExporter},
		{"/log.tsv", exporter.LogTSVExporter},
		{"/log.csv", exporter.LogCSVExporter},
		{"/log.json", exporter.LogJsonExporter},
		{"/metrics", exporter.MetricsExporter},
		{"/healthz", exporter.HealthzExporter},
	}

	for _, tt := range benchmarks {
		b.Run(tt.Path, func(b *testing.B) {
			s := testutil.NewStoreWithLog(b)
			defer s.Close()

			h := tt.Exporter(s)

			r := httptest.NewRequest("GET", tt.Path, nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h(httptest.NewRecorder(), r)
			}
		})
	}
}
