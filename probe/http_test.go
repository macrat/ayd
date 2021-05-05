package probe_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestHTTPProbe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{server.URL + "/ok", store.STATUS_HEALTHY, `200 OK`, ""},
		{server.URL + "/redirect/ok", store.STATUS_HEALTHY, `200 OK`, ""},
		{server.URL + "/error", store.STATUS_FAILURE, `500 Internal Server Error`, ""},
		{server.URL + "/redirect/error", store.STATUS_FAILURE, `500 Internal Server Error`, ""},
		{server.URL + "/redirect/loop", store.STATUS_FAILURE, `Get "/redirect/loop": redirect loop detected`, ""},
		{strings.Replace(server.URL, "http", "http-get", 1) + "/only/get", store.STATUS_HEALTHY, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-post", 1) + "/only/post", store.STATUS_HEALTHY, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-head", 1) + "/only/head", store.STATUS_HEALTHY, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-options", 1) + "/only/options", store.STATUS_HEALTHY, `200 OK`, ""},
		{server.URL + "/slow-page", store.STATUS_UNKNOWN, `probe timed out`, ""},
	})

	AssertTimeout(t, server.URL)
}

func BenchmarkHTTPProbe(b *testing.B) {
	server := RunDummyHTTPServer()
	defer server.Close()

	p := testutil.NewProbe(b, server.URL+"/ok")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
