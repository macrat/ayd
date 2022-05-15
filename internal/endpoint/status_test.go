package endpoint_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
)

func TestStatusHTMLEndpoint(t *testing.T) {
	AssertEndpoint(t, "/status.html", "./testdata/status.html", `Reported by Ayd \(.+\)`)
	AssertEndpoint(t, "/", "./testdata/status.html", `Reported by Ayd \(.+\)`)
}

func TestStatusTextEndpoint(t *testing.T) {
	tests := []struct {
		URL  string
		File string
	}{
		{"/status.txt", "./testdata/status.unicode.txt"},
		{"/status.txt?charset=unicode", "./testdata/status.unicode.txt"},
		{"/status.txt?charset=ascii", "./testdata/status.ascii.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.URL, func(t *testing.T) {
			AssertEndpoint(t, tt.URL, tt.File, ` *Reported by Ayd \(.+\)`)
		})
	}

	t.Run("/status.txt?charset=what", func(t *testing.T) {
		srv := testutil.StartTestServer(t)
		defer srv.Close()

		u := srv.URL + "/status.txt?charset=what"
		if resp, err := srv.Client().Get(u); err != nil {
			t.Errorf("failed to get %s: %s", u, err)
		} else if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("unexpected status: %s", resp.Status)
		}
	})
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
