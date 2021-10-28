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

	for _, x := range []string{"a", "b", "c"} {
		u := &url.URL{Scheme: "http", Host: x + ".example.com"}
		s.ActivateTarget(u, u)
	}

	return httptest.NewServer(endpoint.New(s))
}
