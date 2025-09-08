package main_test

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/cmd/ayd"
	"github.com/macrat/ayd/internal/testutil"
)

//go:embed testdata/log.csv
var testLogCSV string

//go:embed testdata/log.json
var testLogJson string

//go:embed testdata/log.ltsv
var testLogLtsv string

//go:embed testdata/log.xlsx
var testLogXlsx []byte

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
			testutil.DummyLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"-c"},
			testutil.DummyLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"--csv", "-"},
			testutil.DummyLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"-c", "../../internal/testutil/testdata/test.log"},
			testutil.DummyLog,
			testLogCSV,
			"",
			0,
		},
		{
			[]string{"-j"},
			testutil.DummyLog,
			testLogJson,
			"",
			0,
		},
		{
			[]string{"--json", "-o", "-"},
			testutil.DummyLog,
			testLogJson,
			"",
			0,
		},
		{
			[]string{"-l"},
			testutil.DummyLog,
			testLogLtsv,
			"",
			0,
		},
		{
			[]string{"-j", "-c"},
			``,
			"",
			"error: flags for output format can not use multiple in the same time.\n",
			2,
		},
		{
			[]string{"-c", "./testdata/no-such-file"},
			``,
			"",
			"error: failed to open input log file: .*\n",
			1,
		},
		{
			[]string{"-h"},
			``,
			main.ConvHelp,
			"",
			0,
		},
		{
			[]string{"--no-such-option"},
			``,
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

			if diff := cmp.Diff(tt.stdout, stdout.String()); diff != "" {
				t.Errorf("unexpected stdout\n%s", diff)
			}

			if ok, _ := regexp.Match("^"+tt.stderr+"$", stderr.Bytes()); !ok {
				t.Errorf("unexpected stderr\nexpected: %s\n but got: %s", tt.stderr, stderr.String())
			}
		})
	}

	t.Run("write-file", func(t *testing.T) {
		stdin := strings.NewReader(testutil.DummyLog)
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
		if diff := cmp.Diff(testLogCSV, string(output)); diff != "" {
			t.Errorf("%s", diff)
		}
	})
}

func TestConvCommand_Run_xlsx(t *testing.T) {
	stdin := strings.NewReader(testutil.DummyLog)
	output := bytes.NewBuffer(nil)
	cmd := main.ConvCommand{stdin, output, output}

	temp := filepath.Join(t.TempDir(), "output.xlsx")

	code := cmd.Run([]string{"ayd", "conv", "-x", "../../internal/testutil/testdata/test.log", "./testdata/additional.log", "-o", temp})
	defer func() {
		t.Logf("output:\n%s", output.String())
	}()

	if code != 0 {
		t.Fatalf("expected exit code is 0 but got %d", code)
	}

	actual, err := os.Open(temp)
	if err != nil {
		t.Fatalf("failed to open output file: %s", err)
	}
	defer actual.Close()
	as, _ := actual.Stat()

	if !testutil.XlsxEqual(actual, int64(as.Size()), bytes.NewReader(testLogXlsx), int64(len(testLogXlsx))) {
		t.Errorf("output file is different from snapshot.\nplease check testdata/actual/log.xlsx")

		if err = os.MkdirAll("./testdata/actual", 0755); err != nil {
			t.Fatalf("failed to create testdata/actual: %s", err)
		}
		actual, err := os.Open(temp)
		if err != nil {
			t.Fatalf("failed to open output file: %s", err)
		}
		defer actual.Close()
		out, err := os.Create("testdata/actual/log.xlsx")
		if err != nil {
			t.Fatalf("failed to create actual file: %s", err)
		}
		if _, err = io.Copy(out, actual); err != nil {
			t.Errorf("failed to write testdata/actual/log.xlsx: %s", err)
		}
	}
}
