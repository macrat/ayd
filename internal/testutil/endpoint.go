package testutil

import (
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/internal/endpoint"
	api "github.com/macrat/ayd/lib-ayd"
)

func StartTestServer(t testing.TB) *httptest.Server {
	t.Helper()

	s := NewStoreWithLog(t)
	t.Cleanup(func() {
		s.Close()
	})

	for _, x := range []string{"a", "b", "c"} {
		u := &api.URL{Scheme: "http", Host: x + ".example.com"}
		s.ActivateTarget(u, u)
	}

	return httptest.NewServer(endpoint.New(s))
}
