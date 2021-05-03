// +build windows

package probe_test

import (
	"strings"
	"testing"

	"github.com/macrat/ayd/store"
)

func TestTCPProbe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{strings.Replace(server.URL, "http://", "tcp://", 1), store.STATUS_HEALTHY, `(127\.0\.0\.1|\[::1\]):[0-9]+ -> (127\.0\.0\.1|\[::1\]):[0-9]+`, ""},
		{"tcp://localhost:56789", store.STATUS_FAILURE, `dial tcp (127\.0\.0\.1|\[::1\]):56789: connectex: No connection could be made because the target machine actively refused it.`, ""},
		{"tcp://localhost", store.STATUS_UNKNOWN, ``, "TCP target's port number is required"},
	})

	AssertTimeout(t, strings.Replace(server.URL, "http://", "tcp://", 1))
}
