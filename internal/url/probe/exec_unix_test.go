//go:build linux || darwin
// +build linux darwin

package probe_test

import (
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestExecuteProbe_unix(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"exec:./testdata/no-such-script", api.StatusUnknown, ``, `exec: "./testdata/no-such-script": stat ./testdata/no-such-script: no such file or directory`},
		{"exec:./testdata/no-permission.sh", api.StatusUnknown, ``, `exec: "./testdata/no-permission.sh": permission denied`},
		{"exec:no-such-command", api.StatusUnknown, ``, `exec: "no-such-command": executable file not found in \$PATH`},
	}, 5)
}
