package probe_test

import (
	"strings"
	"testing"

	"github.com/macrat/ayd/store"
)

func TestTCPProbe(t *testing.T) {
	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{strings.Replace(server.URL, "http://", "tcp:", 1), store.STATUS_HEALTHY, `(127\.0\.0\.1|\[::1\]):[0-9]+ -> (127\.0\.0\.1|\[::1\]):[0-9]+`},
		{"tcp:localhost:56789", store.STATUS_FAILURE, `dial tcp (127\.0\.0\.1|\[::1\]):56789: connect: connection refused`},
	})
}
