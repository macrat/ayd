package ayd_test

import (
	"net/url"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
	"github.com/macrat/ayd/lib-ayd"
)

func TestFetch(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %s", err)
	}

	_, err = ayd.Fetch(u)
	if err != nil {
		t.Fatalf("failed to fetch: %s", err)
	}
}
