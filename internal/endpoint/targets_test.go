package endpoint_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
)

func TestTargetsTextEndpoint(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	expected := "http://a.example.com\nhttp://b.example.com\nhttp://c.example.com\n"

	if resp, err := srv.Client().Get(srv.URL + "/targets.txt"); err != nil {
		t.Errorf("failed to get /targets.txt: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	} else if raw, err := io.ReadAll(resp.Body); err != nil {
		t.Errorf("failed to read response: %s", err)
	} else if string(raw) != expected {
		t.Errorf("unexpected response\n--- expected --\n%s\n--- actual ---\n%s", expected, string(raw))
	}
}

func TestTargetsJSONEndpoint(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	expected := `["http://a.example.com","http://b.example.com","http://c.example.com"]` + "\n"

	if resp, err := srv.Client().Get(srv.URL + "/targets.json"); err != nil {
		t.Errorf("failed to get /targets.json: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	} else if raw, err := io.ReadAll(resp.Body); err != nil {
		t.Errorf("failed to read response: %s", err)
	} else if string(raw) != expected {
		t.Errorf("unexpected response\n--- expected --\n%s\n--- actual ---\n%s", expected, string(raw))
	}
}
