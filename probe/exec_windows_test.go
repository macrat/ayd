// +build windows

package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{`exec:stub\test.bat?message=hello&code=0`, store.STATUS_HEALTHY, "hello\n"},
		{`exec:stub\test.bat?message=world&code=1`, store.STATUS_FAILURE, "world\n"},
		{`exec:stub\no-such-script`, store.STATUS_UNKNOWN, `exec: "stub\\\\no-such-script": file does not exist`},
		{"exec:no-such-command", store.STATUS_UNKNOWN, `exec: "no-such-command": executable file not found in %PATH%`},
		{"exec:sleep#10", store.STATUS_UNKNOWN, `timeout`},
	})
}
