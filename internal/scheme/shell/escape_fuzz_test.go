//go:build linux || darwin
// +build linux darwin

package shell_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/macrat/ayd/internal/scheme/shell"
)

func FuzzEscape(f *testing.F) {
	f.Add(`hello world`)
	f.Add(`$this is "a" 'test'`)
	f.Add(`hello \ world & abc`)
	f.Add(`abc <def> ghi`)
	f.Add(`-E ignore flag`)

	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) || strings.ContainsRune(s, '\x00') {
			t.SkipNow()
		}

		escaped := shell.Escape(s)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		output, err := exec.CommandContext(ctx, "/bin/sh", "-c", "echo -- "+escaped).CombinedOutput()
		if err != nil {
			t.Log(string(output))
			t.Fatalf("failed to execute shell: %s", err)
		}

		output = output[3 : len(output)-1] // drop "-- " and newline

		if string(output) != s {
			t.Errorf("input: [%s], escaped: [%s], shell unescaped: [%s]", s, escaped, string(output))
		}
	})
}
