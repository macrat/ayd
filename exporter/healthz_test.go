package exporter_test

import (
	"net/http"
	"testing"
)

func TestHealthzExporter(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/healthz"); err != nil {
		t.Errorf("failed to get /healthz: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
