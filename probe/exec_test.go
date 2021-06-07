package probe_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/testutil"
)

func TestExecuteProbe_unknownError(t *testing.T) {
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

	p := testutil.NewProbe(t, "exec:"+file)

	if err := os.Chmod(file, 0000); err != nil {
		t.Fatalf("failed to change permission of test file: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		rs := testutil.RunCheck(ctx, p)
		if rs[0].Status != api.StatusUnknown || !strings.Contains(rs[0].Message, "permission denied") {
			t.Errorf("unexpected result:\n%s", rs[0])
		}
	}

	if err := os.Remove(file); err != nil {
		t.Fatalf("failed to remove test file: %s", err)
	}

	rs := testutil.RunCheck(ctx, p)
	if rs[0].Status != api.StatusUnknown || (!strings.Contains(rs[0].Message, "no such file or directory") && !strings.Contains(rs[0].Message, "file does not exist")) {
		t.Errorf("unexpected result:\n%s", rs[0])
	}
}

func BenchmarkExecuteProbe(b *testing.B) {
	p := testutil.NewProbe(b, "exec:echo#hello-world")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
