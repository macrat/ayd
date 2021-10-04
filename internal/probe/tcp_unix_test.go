//go:build linux || darwin
// +build linux darwin

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
	})

	AssertTimeout(t, strings.Replace(server.URL, "http://", "tcp://", 1))
}
