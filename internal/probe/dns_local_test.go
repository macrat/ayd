//go:build !githubci
// +build !githubci

package probe_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestDNSProbe_local(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dns:of-course-definitely-no-such-host", api.StatusFailure, `lookup of-course-definitely-no-such-host: not found on .+`, ""},
		{"dns://8.8.8.8/of-course-definitely-no-such-host", api.StatusFailure, `lookup of-course-definitely-no-such-host: not found on 8\.8\.8\.8`, ""},
	})
}
