package ayd_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/macrat/ayd/internal/testutil"
	"github.com/macrat/ayd/lib-ayd"
)

func ExampleFetch() {
	aydURL, _ := url.Parse("http://localhost:9000")

	// fetch status from Ayd server
	resp, err := ayd.Fetch(aydURL)
	if err != nil {
		panic(err)
	}

	// parse status of all targets
	allRecords, err := resp.AllRecords()
	if err != nil {
		panic(err)
	}

	// show targets status
	for _, targetRecords := range allRecords {
		record := targetRecords[len(targetRecords)-1]
		fmt.Println(record.Target.String(), ":", record.Status)
	}
}

func TestFetch(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %s", err)
	}

	resp, err := ayd.Fetch(u)
	if err != nil {
		t.Fatalf("failed to fetch: %s", err)
	}

	_, err = resp.AllRecords()
	if err != nil {
		t.Fatalf("failed to parse records: %s", err)
	}
}
