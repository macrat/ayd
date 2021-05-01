package exporter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestJSONExporter(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	x := make(map[string]interface{})

	if resp, err := srv.Client().Get(srv.URL + "/status.json"); err != nil {
		t.Errorf("failed to get /status.json: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	} else if raw, err := io.ReadAll(resp.Body); err != nil {
		t.Errorf("failed to read response: %s", err)
	} else if err := json.Unmarshal(raw, &x); err != nil {
		t.Errorf("failed to parse response: %s", err)
	}
}
