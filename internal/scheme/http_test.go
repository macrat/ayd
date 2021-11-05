package scheme_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
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
		{strings.Replace(server.URL, "http", "http-connect", 1) + "/only/connect", api.StatusHealthy, `proto=HTTP/1\.1 length=0 status=200_OK`, ""},
		{server.URL + "/slow-page", api.StatusFailure, `probe timed out`, ""},
		{"http://localhost:54321", api.StatusFailure, `(127\.0\.0\.1|\[::1\]):54321: connection refused`, ""},
	}, 5)

	AssertTimeout(t, server.URL)

	for _, tt := range []string{"unknown-method", ""} {
		u := "http-" + tt + "://localhost"
		t.Run(u, func(t *testing.T) {
			expected := `HTTP "` + strings.ToUpper(tt) + `" method is not supported. Please use GET, HEAD, POST, OPTIONS, or CONNECT.`
			_, err := scheme.NewProbe(u)
			if err == nil {
				t.Errorf("expected error but got nil")
			} else if err.Error() != expected {
				t.Errorf("unexpected error:\nexpected: %s\n but got: %s", expected, err)
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
