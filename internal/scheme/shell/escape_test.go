package shell_test

import (
	"testing"

	"github.com/macrat/ayd/internal/scheme/shell"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{`hello world`, `"hello world"`},
		{`$this is a "test"`, `"\$this is a \"test\""`},
		{`oh\&no`, `"oh\\&no"`},
		{`echo < this > is (test)!`, `"echo < this > is (test)!"`},
	}

	for _, tt := range tests {
		output := shell.Escape(tt.Input)
		if tt.Output != output {
			t.Errorf("[%s] should be [%s] but got [%s]", tt.Input, tt.Output, output)
		}
	}
}
