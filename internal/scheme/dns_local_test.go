//go:build !githubci
// +build !githubci

package scheme_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestDNSScheme_local(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dns:of-course-definitely-no-such-host", api.StatusFailure, `lookup of-course-definitely-no-such-host: .+`, ""},
		{"dns://8.8.8.8/of-course-definitely-no-such-host", api.StatusFailure, `lookup of-course-definitely-no-such-host: .+`, ""},
	}, 10)
}
