package probe_test

import (
	"strings"
	"testing"

	"github.com/macrat/ayd/store"
)

func TestHTTPProbe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{server.URL + "/ok", store.STATUS_HEALTHY, `200 OK`},
		{server.URL + "/redirect/ok", store.STATUS_HEALTHY, `200 OK`},
		{server.URL + "/error", store.STATUS_FAILURE, `500 Internal Server Error`},
		{server.URL + "/redirect/error", store.STATUS_FAILURE, `500 Internal Server Error`},
		{server.URL + "/redirect/loop", store.STATUS_FAILURE, `Get "/redirect/loop": redirect loop detected`},
		{strings.Replace(server.URL, "http", "http-get", 1) + "/only/get", store.STATUS_HEALTHY, `200 OK`},
		{strings.Replace(server.URL, "http", "http-post", 1) + "/only/post", store.STATUS_HEALTHY, `200 OK`},
		{strings.Replace(server.URL, "http", "http-head", 1) + "/only/head", store.STATUS_HEALTHY, `200 OK`},
		{strings.Replace(server.URL, "http", "http-options", 1) + "/only/options", store.STATUS_HEALTHY, `200 OK`},
		{server.URL + "/slow-page", store.STATUS_UNKNOWN, `timed out or interrupted`},
	})

	AssertTimeout(t, server.URL)
}
