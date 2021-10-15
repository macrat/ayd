package endpoint_test

import (
	"net/http"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
)

func TestMetricsEndpoint(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/metrics"); err != nil {
		t.Errorf("failed to get /metrics: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
