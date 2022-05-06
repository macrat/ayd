package endpoint_test

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/testutil"
)

func TestIncidentsHTMLEndpoint(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/incidents.html"); err != nil {
		t.Errorf("failed to get /incidents.html: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}

	resp, err := srv.Client().Get(srv.URL + "/incidents.html")

	if err != nil {
		t.Errorf("failed to get /incidents.html: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}

	result := string(regexp.MustCompile(`Reported by Ayd \(.+\)`).ReplaceAll(body, []byte("Reported by Ayd (-- CURRENT TIME MASKED --)")))
	result = strings.ReplaceAll(result, "\r\n", "\n")

	if diff := cmp.Diff(readTestFile(t, "./testdata/incidents.html"), result); diff != "" {
		t.Errorf(diff)
	}
}
