package exporter_test

import (
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/internal/exporter"
	"github.com/macrat/ayd/internal/testutil"
)

func BenchmarkStatusHTMLExporter(b *testing.B) {
	s := testutil.NewStoreWithLog(b)
	defer s.Close()

	h := exporter.StatusHTMLExporter(s)

	r := httptest.NewRequest("GET", "/status.html", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkStatusTextExporter(b *testing.B) {
	s := testutil.NewStoreWithLog(b)
	defer s.Close()

	h := exporter.StatusTextExporter(s)

	r := httptest.NewRequest("GET", "/status.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkStatusJSONExporter(b *testing.B) {
	s := testutil.NewStoreWithLog(b)
	defer s.Close()

	h := exporter.StatusJSONExporter(s)

	r := httptest.NewRequest("GET", "/status.json", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkMetricsExporter(b *testing.B) {
	s := testutil.NewStoreWithLog(b)
	defer s.Close()

	h := exporter.MetricsExporter(s)

	r := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkHealthzExporter(b *testing.B) {
	s := testutil.NewStoreWithLog(b)
	defer s.Close()

	h := exporter.HealthzExporter(s)

	r := httptest.NewRequest("GET", "/healthz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}
