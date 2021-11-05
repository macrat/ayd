//go:build !githubci
// +build !githubci

package scheme_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestHTTPScheme_local(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"http://of-course-no-such-host.local", api.StatusUnknown, "lookup of-course-no-such-host.local: not found(| on .+)", ""},
	}, 5)
}
