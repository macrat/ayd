package main_test

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd"
)

//go:embed internal/testutil/testdata/test.log
var testLog string

//go:embed testdata/log.csv
var testLogCSV string

//go:embed testdata/log.json
var testLogJson string

//go:embed testdata/log.ltsv
var testLogLtsv string

func TestConvCommand_Run(t *testing.T) {
	tests := []struct {
		args   []string
		stdin  string
		stdout string
		stderr string
		code   int
	}{
		{
			[]string{},
			testLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"-c"},
			testLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"--csv", "-"},
			testLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"-c", "./internal/testutil/testdata/test.log"},
			testLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"-j"},
			testLog,
			testLogJson,
			"",
			0,
		},
		{
			[]string{"--json", "-o", "-"},
			testLog,
			testLogJson,
			"",
			0,
		},
		{
			[]string{"-l"},
			testLog,
			testLogLtsv,
			"",
			0,
		},
		{
			[]string{"-j", "-c"},
			testLog,
			"",
			"error: flags for output format can not use multiple in the same time.\n",
			2,
		},
		{
			[]string{"-c", "./testdata/no-such-file"},
			testLog,
			"",
			"error: failed to open input log file: .*\n",
			1,
		},
		{
			[]string{"-h"},
			testLog,
			main.ConvHelp,
			"",
			0,
		},
		{
			[]string{"--no-such-option"},
			testLog,
			"",
			"unknown flag: --no-such-option\n\nPlease see `ayd conv -h` for more information.\n",
			2,
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			stdin := strings.NewReader(tt.stdin)
			stdout := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)
			cmd := main.ConvCommand{stdin, stdout, stderr}

			if code := cmd.Run(append([]string{"ayd", "conv"}, tt.args...)); tt.code != code {
				t.Errorf("expected exit code is %d but got %d", tt.code, code)
			}

			if diff := cmp.Diff(stdout.String(), tt.stdout); diff != "" {
				t.Errorf("unexpected stdout\n%s", diff)
			}

			if ok, _ := regexp.Match("^"+tt.stderr+"$", stderr.Bytes()); !ok {
				t.Errorf("unexpected stderr\nexpected: %s\n but got: %s", tt.stderr, stderr.String())
			}
		})
	}

	t.Run("write-file", func(t *testing.T) {
		stdin := strings.NewReader(testLog)
		stdout := bytes.NewBuffer(nil)
		stderr := bytes.NewBuffer(nil)
		cmd := main.ConvCommand{stdin, stdout, stderr}

		fpath := filepath.Join(t.TempDir(), "log.csv")

		if code := cmd.Run([]string{"ayd", "conv", "-o", fpath}); code != 0 {
			t.Fatalf("unexpected exit code: %d", code)
		}

		if len(stdout.Bytes()) > 0 {
			t.Errorf("unexpected stdout\n%s", stdout.String())
		}

		if len(stderr.Bytes()) > 0 {
			t.Errorf("unexpected stderr\n%s", stderr.String())
		}

		output, err := os.ReadFile(fpath)
		if err != nil {
			t.Fatalf("failed to read output file: %s", err)
		}
		if diff := cmp.Diff(string(output), testLogCSV); diff != "" {
			t.Errorf(diff)
		}
	})
}
