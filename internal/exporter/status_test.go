package exporter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
)

func TestStatusHTMLExporter(t *testing.T) {
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

func TestStatusTextExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	tests := []string{
		"/status.txt",
		"/status.txt?charset=unicode",
		"/status.txt?charset=ascii",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			u := srv.URL + tt
			if resp, err := srv.Client().Get(u); err != nil {
				t.Errorf("failed to get %s: %s", u, err)
			} else if resp.StatusCode != http.StatusOK {
				t.Errorf("unexpected status: %s", resp.Status)
			}
		})
	}
}

func TestStatusJSONExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
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
