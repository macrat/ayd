package endpoint_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
)

func TestStatusHTMLEndpoint(t *testing.T) {
	AssertEndpoint(t, "/status.html", "./testdata/status.html", `Reported by Ayd \(.+\)|[0-9] years? ago`)
	AssertEndpoint(t, "/", "./testdata/status.html", `Reported by Ayd \(.+\)|[0-9] years? ago`)
}

func TestStatusTextEndpoint(t *testing.T) {
	AssertEndpoint(t, "/status.txt", "./testdata/status.txt", ` *Reported by Ayd \(.+\)`)
}

func TestStatusJSONEndpoint(t *testing.T) {
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
