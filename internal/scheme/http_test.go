package scheme_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestHTTPScheme_Probe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{server.URL + "/ok", api.StatusHealthy, `200 OK`, ""},
		{server.URL + "/redirect/ok", api.StatusHealthy, `200 OK`, ""},
		{server.URL + "/error", api.StatusFailure, `500 Internal Server Error`, ""},
		{server.URL + "/redirect/error", api.StatusFailure, `500 Internal Server Error`, ""},
		{server.URL + "/redirect/loop", api.StatusFailure, `Get "/redirect/loop": redirect loop detected`, ""},
		{strings.Replace(server.URL, "http", "http-get", 1) + "/only/get", api.StatusHealthy, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-post", 1) + "/only/post", api.StatusHealthy, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-head", 1) + "/only/head", api.StatusHealthy, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-options", 1) + "/only/options", api.StatusHealthy, `200 OK`, ""},
		{strings.Replace(server.URL, "http", "http-connect", 1) + "/only/connect", api.StatusHealthy, `200 OK`, ""},
		{server.URL + "/slow-page", api.StatusFailure, `probe timed out`, ""},
		{"http://localhost:54321/", api.StatusFailure, `(127\.0\.0\.1|\[::1\]):54321: connection refused`, ""},
	}, 5)

	AssertTimeout(t, server.URL)

	for _, tt := range []string{"unknown-method", ""} {
		u := "http-" + tt + "://localhost"
		t.Run(u, func(t *testing.T) {
			expected := `HTTP "` + strings.ToUpper(tt) + `" method is not supported. Please use GET, HEAD, POST, OPTIONS, or CONNECT.`
			_, err := scheme.NewProber(u)
			if err == nil {
				t.Errorf("expected error but got nil")
			} else if err.Error() != expected {
				t.Errorf("unexpected error:\nexpected: %s\n but got: %s", expected, err)
			}
		})
	}
}

func TestHTTPScheme_Alert(t *testing.T) {
	t.Parallel()

	log := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log = append(log, r.URL.String())
	}))

	a, err := scheme.NewAlerter(server.URL)
	if err != nil {
		t.Fatalf("failed to prepare HTTPScheme: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}

	a.Alert(ctx, r, api.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:  api.StatusFailure,
		Latency: 123456 * time.Microsecond,
		Target:  &api.URL{Scheme: "dummy", Fragment: "hello"},
		Message: "hello world",
	})

	if len(r.Records) != 1 {
		t.Errorf("unexpected number of records\n%v", r.Records)
	}

	if len(log) != 1 {
		t.Fatalf("unexpected number of request in the log\n%s", log)
	}

	expected := `/?ayd_latency=123.456&ayd_message=hello+world&ayd_status=FAILURE&ayd_target=dummy%3A%23hello&ayd_time=2021-01-02T15%3A04%3A05Z`
	if log[0] != expected {
		t.Errorf("unexpected request URL\nexpected: %s\n but got: %s", expected, log[0])
	}
}

func BenchmarkHTTPScheme(b *testing.B) {
	server := RunDummyHTTPServer()
	defer server.Close()

	p := testutil.NewProber(b, server.URL+"/ok")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Probe(ctx, r)
	}
}
