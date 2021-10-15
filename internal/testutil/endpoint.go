package testutil

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/macrat/ayd/internal/endpoint"
)

func StartTestServer(t testing.TB) *httptest.Server {
	t.Helper()

	s := NewStoreWithLog(t)
	t.Cleanup(func() {
		s.Close()
	})

	s.AddTarget(&url.URL{Scheme: "http", Host: "a.example.com"})
	s.AddTarget(&url.URL{Scheme: "http", Host: "b.example.com"})
	s.AddTarget(&url.URL{Scheme: "http", Host: "c.example.com"})

	return httptest.NewServer(endpoint.New(s))
}
