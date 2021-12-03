//go:build !githubci
// +build !githubci

package scheme_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestFTPProbe_local(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"ftp://of-course-no-such-host.local:21021/", api.StatusUnknown, `lookup of-course-no-such-host.local: not found(| on .+)`, ""},
	}, 5)
}
