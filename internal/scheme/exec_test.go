package scheme_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestExecScheme_Probe(t *testing.T) {
	if runtime.GOOS != "windows" {
		// This test in windows sometimes be fail if enable parallel.
		// Maybe it's because of the timing to unset path to testdata/dos_polyfill.
		t.Parallel()
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+string(filepath.ListSeparator)+filepath.Join(cwd, "testdata", "dos_polyfill"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})

	cwd = filepath.ToSlash(cwd)

	AssertProbe(t, []ProbeTest{
		{"exec:./testdata/test?message=hello&code=0", api.StatusHealthy, "hello\n---\nexit_code: 0", ""},
		{"exec:./testdata/test?message=world&code=1", api.StatusFailure, "world\n---\nexit_code: 1", ""},
		{"exec:./testdata/test?message=::foo::bar&code=1", api.StatusFailure, "---\nexit_code: 1\nfoo: bar", ""},
		{"exec:" + path.Join(cwd, "testdata/test") + "?message=hello&code=0", api.StatusHealthy, "hello\n---\nexit_code: 0", ""},
		{"exec:sleep#10", api.StatusFailure, `probe timed out`, ""},
		{"exec:echo#::status::unknown", api.StatusUnknown, "---\nexit_code: 0", ""},
		{"exec:echo#::status::failure", api.StatusFailure, "---\nexit_code: 0", ""},
	}, 5)

	AssertTimeout(t, "exec:echo")
}

func TestExecScheme_Probe_unknownError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.bat")

	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %s", err)
	}
	if err := f.Chmod(0766); err != nil {
		t.Fatalf("failed to change permission of test file: %s", err)
	}
	f.Close()

	p := testutil.NewProber(t, "exec:"+file)

	if err := os.Chmod(file, 0000); err != nil {
		t.Fatalf("failed to change permission of test file: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		rs := testutil.RunProbe(ctx, p)
		if rs[0].Status != api.StatusUnknown || !strings.Contains(rs[0].Message, "permission denied") {
			t.Errorf("unexpected result:\n%s", rs[0])
		}
	}

	if err := os.Remove(file); err != nil {
		t.Fatalf("failed to remove test file: %s", err)
	}

	rs := testutil.RunProbe(ctx, p)
	if rs[0].Status != api.StatusUnknown || (!strings.Contains(rs[0].Message, "no such file or directory") && !strings.Contains(rs[0].Message, "file does not exist") && !strings.Contains(rs[0].Message, "The system cannot find the file specified.")) {
		t.Errorf("unexpected result:\n%s", rs[0])
	}
}

func BenchmarkExecScheme(b *testing.B) {
	p := testutil.NewProber(b, "exec:echo#hello-world")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Probe(ctx, r)
	}
}
