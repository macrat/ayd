//go:build windows
// +build windows

package scheme_test

import (
	"strings"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestTCPScheme_errors(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{"tcp://localhost:56789", api.StatusFailure, `dial tcp (127\.0\.0\.1|\[::1\]):56789: connectex: No connection could be made because the target machine actively refused it.`, ""},
	}, 5)

	AssertTimeout(t, strings.Replace(server.URL, "http://", "tcp://", 1))
}
