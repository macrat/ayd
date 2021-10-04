//go:build !githubci
// +build !githubci

package probe_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestHTTPProbe_dnsErrors(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"http://of-course-no-such-host.local", api.StatusUnknown, "lookup of-course-no-such-host.local(:| ).+", ""},
	})
}
