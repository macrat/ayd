// +build !windows

package probe_test

import (
	"os"
	"path"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	AssertProbe(t, []ProbeTest{
		{"exec:./testdata/test.sh?message=hello&code=0", api.StatusHealthy, "hello", ""},
		{"exec:./testdata/test.sh?message=world&code=1", api.StatusFailure, "world", ""},
		{"exec:" + path.Join(cwd, "testdata/test.sh") + "?message=hello&code=0", api.StatusHealthy, "hello", ""},
		{"exec:echo#%0Ahello%0Aworld%0A%0A", api.StatusHealthy, "hello\nworld", ""},
		{"exec:./testdata/no-such-script", api.StatusUnknown, ``, `exec: "./testdata/no-such-script": stat ./testdata/no-such-script: no such file or directory`},
		{"exec:./testdata/no-permission.sh", api.StatusUnknown, ``, `exec: "./testdata/no-permission.sh": permission denied`},
		{"exec:no-such-command", api.StatusUnknown, ``, `exec: "no-such-command": executable file not found in \$PATH`},
		{"exec:sleep#10", api.StatusFailure, `probe timed out`, ""},
		{"exec:echo#::status::unknown", api.StatusUnknown, ``, ""},
		{"exec:echo#::status::failure", api.StatusFailure, ``, ""},
	})

	AssertTimeout(t, "exec:echo")
}
