package exporter_test

import (
	"net/http"
	"testing"

	"github.com/macrat/ayd/testutil"
)

func TestTextExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/status.txt"); err != nil {
		t.Errorf("failed to get /status.txt: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
