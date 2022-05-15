//go:build windows
// +build windows

package scheme_test

import (
	"testing"

	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestExecScheme_Probe_windows(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"exec:./testdata/no-such-script", api.StatusUnknown, ``, `exec: ".\\\\testdata\\\\no-such-script": file does not exist`},
		{"exec:no-such-command", api.StatusUnknown, ``, `exec: "no-such-command": executable file not found in %PATH%`},
	}, 5)

	t.Run("normalize-path", func(t *testing.T) {
		tests := []struct {
			From string
			To   string
		}{
			{`./testdata/test.bat`, `./testdata/test.bat`},
			{`.\testdata\test.bat`, `./testdata/test.bat`},
		}

		for _, tt := range tests {
			p, err := scheme.NewExecScheme(&api.URL{Scheme: "exec", Opaque: tt.From})
			if err != nil {
				t.Errorf("%s: failed to create probe: %s", tt.From, err)
			}

			if p.Target().Opaque != tt.To {
				t.Errorf("%s: unexpected path: %s", tt.From, p.Target().Opaque)
			}
		}
	})
}
