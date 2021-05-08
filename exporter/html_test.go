package exporter_test

import (
	"net/http"
	"testing"

	"github.com/macrat/ayd/testutil"
)

func TestHTMLExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/status.html"); err != nil {
		t.Errorf("failed to get /status.html: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}

	if resp, err := srv.Client().Get(srv.URL + "/"); err != nil {
		t.Errorf("failed to get /: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
