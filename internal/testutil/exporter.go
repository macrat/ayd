package testutil

import (
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/internal/exporter"
)

func StartTestServer(t testing.TB) *httptest.Server {
	t.Helper()

	s := NewStoreWithLog(t)
	t.Cleanup(func() {
		s.Close()
	})
	return httptest.NewServer(exporter.New(s))
}
