// +build windows

package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{`exec:testdata\test.bat?message=hello&code=0`, store.STATUS_HEALTHY, "hello"},
		{`exec:testdata\test.bat?message=world&code=1`, store.STATUS_FAILURE, "world"},
		{"exec:echo#%0Ahello%0Aworld%0A%0A", store.STATUS_HEALTHY, "hello\nworld"},
		{`exec:testdata\no-such-script`, store.STATUS_UNKNOWN, `exec: "testdata\\\\no-such-script": file does not exist`},
		{"exec:no-such-command", store.STATUS_UNKNOWN, `exec: "no-such-command": executable file not found in %PATH%`},
		{"exec:sleep#10", store.STATUS_UNKNOWN, `timeout`},
		{"exec:echo#::status::unknown", store.STATUS_UNKNOWN, ``},
		{"exec:echo#::status::failure", store.STATUS_FAILURE, ``},
	})
}
