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
	report, err := ayd.Fetch(aydURL)
	if err != nil {
		panic(err)
	}

	for target, status := range report.ProbeHistory {
		// show target name
		fmt.Printf("# %s\n", target)

		// show status history
		for _, x := range status.Records {
			fmt.Println(x.Status)
		}
	}
}

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
