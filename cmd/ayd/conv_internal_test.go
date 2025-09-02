package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	"github.com/mattn/go-isatty"
)

func init() {
	CurrentTime = func() time.Time {
		return time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC)
	}
}

func TestConvCommand_TTYWarning(t *testing.T) {
	formats := []struct {
		name       string
		args       []string
		expectWarn bool
	}{
		{"csv_default", []string{}, false},
		{"csv", []string{"-c"}, false},
		{"json", []string{"-j"}, false},
		{"ltsv", []string{"-l"}, false},
		{"xlsx", []string{"-x"}, true},
	}

	combos := []struct {
		name string
		term bool
		cyg  bool
	}{
		{"no_tty", false, false},
		{"tty", true, false},
		{"cygwin", false, true},
		{"tty_cygwin", true, true},
	}

	for _, cmb := range combos {
		for _, f := range formats {
			name := cmb.name + "/" + f.name
			t.Run(name, func(t *testing.T) {
				isTerminal = func(uintptr) bool { return cmb.term }
				isCygwinTerminal = func(uintptr) bool { return cmb.cyg }
				defer func() {
					isTerminal = isatty.IsTerminal
					isCygwinTerminal = isatty.IsCygwinTerminal
				}()

				stdout := bytes.NewBuffer(nil)
				stderr := bytes.NewBuffer(nil)
				cmd := ConvCommand{strings.NewReader(testutil.DummyLog), stdout, stderr}

				code := cmd.Run(append([]string{"ayd", "conv"}, f.args...))

				warn := strings.Contains(stderr.String(), "can not write xlsx format")
				if f.expectWarn && (cmb.term || cmb.cyg) {
					if code != 2 {
						t.Errorf("expect exit 2 but got %d", code)
					}
					if !warn {
						t.Errorf("expected warning, got %s", stderr.String())
					}
				} else {
					if code != 0 {
						t.Errorf("unexpected exit code %d", code)
					}
					if warn {
						t.Errorf("unexpected warning: %s", stderr.String())
					}
				}
			})
		}
	}
}
