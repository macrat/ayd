package exporter_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/testutil"
)

func readTestFile(t *testing.T, file string) string {
	t.Helper()

	f, err := os.Open(file)
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}

	return string(bs)
}

func TestStatusHTMLExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/status.html"); err != nil {
		t.Errorf("failed to get /status.html: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}

	resp, err := srv.Client().Get(srv.URL + "/")

	if err != nil {
		t.Errorf("failed to get /: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}

	result := string(regexp.MustCompile(`Reported By Ayd\? \(.+\)`).ReplaceAll(body, []byte("Reported By Ayd? (-- CURRENT TIME MASKED --)")))

	if diff := cmp.Diff(readTestFile(t, "./testdata/status.html"), result); diff != "" {
		t.Errorf(diff)
	}
}

func TestStatusTextExporter(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

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
			u := srv.URL + tt.URL
			resp, err := srv.Client().Get(u)

			if err != nil {
				t.Errorf("failed to get %s: %s", u, err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("unexpected status: %s", resp.Status)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}

			result := string(regexp.MustCompile(` *Reported by Ayd\? \(.+\)`).ReplaceAll(body, []byte("[[ FOOTER MASKED ]]")))

			if diff := cmp.Diff(readTestFile(t, fmt.Sprintf(tt.File)), result); diff != "" {
				t.Errorf(diff)
			}
		})
	}

	t.Run("/status.txt?charset=what", func(t *testing.T) {
		u := srv.URL + "/status.txt?charset=what"
		if resp, err := srv.Client().Get(u); err != nil {
			t.Errorf("failed to get %s: %s", u, err)
		} else if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("unexpected status: %s", resp.Status)
		}
	})
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
