//go:build !windows
// +build !windows

package probe_test

import (
	"strings"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestTCPProbe_errors(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{"tcp://localhost:56789", api.StatusFailure, `dial tcp (127\.0\.0\.1|\[::1\]):56789: connect: connection refused`, ""},
		{"tcp://localhost", api.StatusUnknown, ``, "TCP target's port number is required"},
	})

	AssertTimeout(t, strings.Replace(server.URL, "http://", "tcp://", 1))
}
