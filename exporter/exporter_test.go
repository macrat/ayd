package exporter_test

import (
	"net/http"
	"testing"

	"github.com/macrat/ayd/testutil"
)

func TestStaticFiles(t *testing.T) {
	srv := testutil.StartTestServer(t)
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
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/not-found"); err != nil {
		t.Errorf("failed to get /not-found: %s", err)
	} else if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
