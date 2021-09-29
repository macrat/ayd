package probe_test

import (
	"context"
	"strings"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/testutil"
)

func TestHTTPProbe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{server.URL + "/ok", api.StatusHealthy, `proto=HTTP/1\.1 length=2 status=200_OK`, ""},
		{server.URL + "/redirect/ok", api.StatusHealthy, `proto=HTTP/1\.1 length=2 status=200_OK`, ""},
		{server.URL + "/error", api.StatusFailure, `proto=HTTP/1\.1 length=5 status=500_Internal_Server_Error`, ""},
		{server.URL + "/redirect/error", api.StatusFailure, `proto=HTTP/1\.1 length=5 status=500_Internal_Server_Error`, ""},
		{server.URL + "/redirect/loop", api.StatusFailure, `Get "/redirect/loop": redirect loop detected`, ""},
		{strings.Replace(server.URL, "http", "http-get", 1) + "/only/get", api.StatusHealthy, `proto=HTTP/1\.1 length=0 status=200_OK`, ""},
		{strings.Replace(server.URL, "http", "http-post", 1) + "/only/post", api.StatusHealthy, `proto=HTTP/1\.1 length=0 status=200_OK`, ""},
		{strings.Replace(server.URL, "http", "http-head", 1) + "/only/head", api.StatusHealthy, `proto=HTTP/1\.1 length=-1 status=200_OK`, ""},
		{strings.Replace(server.URL, "http", "http-options", 1) + "/only/options", api.StatusHealthy, `proto=HTTP/1\.1 length=0 status=200_OK`, ""},
		{server.URL + "/slow-page", api.StatusFailure, `probe timed out`, ""},
		{"http://localhost:54321", api.StatusFailure, `(127\.0\.0\.1|\[::1\]):54321: connection refused`, ""},
	})

	AssertTimeout(t, server.URL)

	for _, tt := range []string{"unknown-method", ""} {
		u := "http-" + tt + "://localhost"
		t.Run(u, func(t *testing.T) {
			_, err := probe.New(u)
			if err == nil {
				t.Errorf("expected error but got nil")
			} else if err.Error() != `HTTP "`+strings.ToUpper(tt)+`" method is not supported. Please use GET, HEAD, POST, or OPTIONS.` {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
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
