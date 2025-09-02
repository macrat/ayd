//go:build container
// +build container

// Container test is a set of tests work with external servers running on Docker compose.
// Please make sure that containers defined in ../../testdata/docker-compose.yml are run, before execute these tests.

package scheme_test

import (
	"net/http"
	"testing"
)

func ResetTestContainer(t *testing.T, scope string) {
	req, err := http.NewRequest(http.MethodDelete, "http://localhost/"+scope, nil)
	if err != nil {
		t.Fatalf("failed to prepare reset request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to reset: %s", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("failed to reset: %s", resp.Status)
	}
}
