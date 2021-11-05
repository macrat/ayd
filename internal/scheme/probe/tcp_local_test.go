//go:build !githubci
// +build !githubci

package probe_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestTCPProbe_local(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"tcp://of-course-no-such-host.local:54321", api.StatusUnknown, "lookup of-course-no-such-host.local: not found(| on .+)", ""},
	}, 5)
}
