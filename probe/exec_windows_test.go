// +build windows

package probe_test

import (
	"testing"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{`exec:./testdata/test.bat?message=hello&code=0`, store.STATUS_HEALTHY, "hello", ""},
		{`exec:./testdata/test.bat?message=world&code=1`, store.STATUS_FAILURE, "world", ""},
		{"exec:echo#%0Ahello%0Aworld%0A%0A", store.STATUS_HEALTHY, "hello\nworld", ""},
		{"exec:./testdata/no-such-script", store.STATUS_UNKNOWN, ``, `exec: ".\testdata\no-such-script": stat .\testdata\no-such-script: no such file or directory`},
		{"exec:no-such-command", store.STATUS_UNKNOWN, ``, `exec: "no-such-command": executable file not found in %PATH%`},
		{"exec:sleep#10", store.STATUS_UNKNOWN, `probe timed out`, ""},
		{"exec:echo#::status::unknown", store.STATUS_UNKNOWN, ``, ""},
		{"exec:echo#::status::failure", store.STATUS_FAILURE, ``, ""},
	})

	AssertTimeout(t, "exec:echo")

	t.Func("normalize-path", func(t *testing.T) {
		tests := struct {
			From string
			To   string
		}{
			{`./testdata/test.bat`, `./testdata/test.bat`},
			{`.\testdata\test.bat`, `./testdata/test.bat`},
		}

		for _, tt := range tests {
			p, err := probe.NewExecProbe(&url.URL{Scheme: "exec", Opaque: tt.From})
			if err != nil {
				t.Errorf("%s: failed to create probe: %s", tt.From, err)
			}

			if p.Target().Opaque != tt.To {
				t.Errorf("%s: unexpected path: %s", tt.From, p.Target().Opaque)
			}
		}
	})
}
