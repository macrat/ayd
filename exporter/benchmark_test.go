package exporter_test

import (
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/store"
)

func BenchmarkHTMLExporter(b *testing.B) {
	s, err := store.New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	h := exporter.HTMLExporter(s)

	r := httptest.NewRequest("GET", "/status.html", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkTextExporter(b *testing.B) {
	s, err := store.New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	h := exporter.TextExporter(s)

	r := httptest.NewRequest("GET", "/status.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkJSONExporter(b *testing.B) {
	s, err := store.New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	h := exporter.JSONExporter(s)

	r := httptest.NewRequest("GET", "/status.json", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkMetricsExporter(b *testing.B) {
	s, err := store.New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	h := exporter.JSONExporter(s)

	r := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkHealthzExporter(b *testing.B) {
	s, err := store.New("./testdata/test.log")
	if err != nil {
		b.Fatalf("failed to open store: %s", err)
	}

	if err = s.Restore(); err != nil {
		b.Fatalf("failed to restore store: %s", err)
	}

	h := exporter.HealthzExporter(s)

	r := httptest.NewRequest("GET", "/healthz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}
