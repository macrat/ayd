package exporter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/macrat/ayd/testutil"
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

func TestStatusUnicodeTextExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/status.unicode.txt"); err != nil {
		t.Errorf("failed to get /status.unicode.txt: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestStatusASCIITextExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/status.ascii.txt"); err != nil {
		t.Errorf("failed to get /status.ascii.txt: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
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
