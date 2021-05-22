package testutil

import (
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/store"
)

func StartTestServer(t testing.TB) *httptest.Server {
	t.Helper()

	s, err := store.New("../exporter/testdata/test.log")
	if err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
	t.Cleanup(func() {
		s.Close()
	})
	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}

	return httptest.NewServer(exporter.New(s))
}
