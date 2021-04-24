// +build !windows

package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"exec:./stub/test.sh?message=hello&code=0", store.STATUS_HEALTHY, "hello"},
		{"exec:./stub/test.sh?message=world&code=1", store.STATUS_FAILURE, "world"},
		{"exec:echo#%0Ahello%0Aworld%0A%0A", store.STATUS_HEALTHY, "hello\nworld"},
		{"exec:./stub/no-such-script", store.STATUS_UNKNOWN, `fork/exec ./stub/no-such-script: no such file or directory`},
		{"exec:./stub/no-permission.sh", store.STATUS_UNKNOWN, `fork/exec ./stub/no-permission.sh: permission denied`},
		{"exec:no-such-command", store.STATUS_UNKNOWN, `exec: "no-such-command": executable file not found in \$PATH`},
		{"exec:sleep#10", store.STATUS_UNKNOWN, `timeout`},
		{"exec:echo#::status::unknown", store.STATUS_UNKNOWN, ``},
		{"exec:echo#::status::failure", store.STATUS_FAILURE, ``},
	})
}
