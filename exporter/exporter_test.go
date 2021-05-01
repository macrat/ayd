package exporter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/store"
)

func StartTestServer(t testing.TB) *httptest.Server {
	t.Helper()

	s, err := store.New("./testdata/test.log")
	if err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
	t.Cleanup(func() {
		s.Close()
	})

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}

	return httptest.NewServer(exporter.New(s))
}

func TestStaticFiles(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/favicon.ico"); err != nil {
		t.Errorf("failed to get /favicon.ico: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}

	if resp, err := srv.Client().Get(srv.URL + "/favicon.svg"); err != nil {
		t.Errorf("failed to get /favicon.svg: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestNotFound(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/not-found"); err != nil {
		t.Errorf("failed to get /not-found: %s", err)
	} else if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
