// +build windows

package probe_test

import (
	"net/url"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
)

func TestExecuteProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{`exec:./testdata/test.bat?message=hello&code=0`, api.StatusHealthy, "hello", ""},
		{`exec:./testdata/test.bat?message=world&code=1`, api.StatusFailure, "world", ""},
		{"exec:echo#%0Ahello%0Aworld%0A%0A", api.StatusHealthy, "hello\nworld", ""},
		{"exec:./testdata/no-such-script", api.StatusUnknown, ``, `exec: ".\\\\testdata\\\\no-such-script": file does not exist`},
		{"exec:no-such-command", api.StatusUnknown, ``, `exec: "no-such-command": executable file not found in %PATH%`},
		{"exec:sleep#10", api.StatusUnknown, `probe timed out`, ""},
		{"exec:echo#::status::unknown", api.StatusUnknown, ``, ""},
		{"exec:echo#::status::failure", api.StatusFailure, ``, ""},
	})

	AssertTimeout(t, "exec:echo")

	t.Run("normalize-path", func(t *testing.T) {
		tests := []struct {
			From string
			To   string
		}{
			{`./testdata/test.bat`, `./testdata/test.bat`},
			{`.\testdata\test.bat`, `./testdata/test.bat`},
		}

		for _, tt := range tests {
			p, err := probe.NewExecuteProbe(&url.URL{Scheme: "exec", Opaque: tt.From})
			if err != nil {
				t.Errorf("%s: failed to create probe: %s", tt.From, err)
			}

			if p.Target().Opaque != tt.To {
				t.Errorf("%s: unexpected path: %s", tt.From, p.Target().Opaque)
			}
		}
	})
}
