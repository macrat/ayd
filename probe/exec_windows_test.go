// +build windows

package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestExecuteProbe(t *testing.T) {
	AssertProbe(t, []ProbeTest{
		{`exec:stub\test.bat?message=hello&code=0`, store.STATUS_HEALTHY, "hello\r\n"},
		{`exec:stub\test.bat?message=world&code=1`, store.STATUS_FAILURE, "world\r\n"},
		{`exec:stub\no-such-script`, store.STATUS_UNKNOWN, `exec: "stub\\\\no-such-script": file does not exist`},
		{"exec:no-such-command", store.STATUS_UNKNOWN, `exec: "no-such-command": executable file not found in %PATH%`},
	})
}
