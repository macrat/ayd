package testutil

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/store"
)

func CopyFile(t testing.TB, source string) (dest string) {
	dest = filepath.Join(t.TempDir(), "test.log")

	srcFile, err := os.Open(source)
	if err != nil {
		t.Fatalf("failed to open source file: %s", err)
	}

	dstFile, err := os.Create(dest)
	if err != nil {
		t.Fatalf("failed to open dest file: %s", err)
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		t.Fatalf("failed to copy file: %s", err)
	}

	return dest
}

func StartTestServer(t testing.TB) *httptest.Server {
	t.Helper()

	s, err := store.New(CopyFile(t, "../exporter/testdata/test.log"))
	if err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
	s.Console = io.Discard
	t.Cleanup(func() {
		s.Close()
	})
	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}

	return httptest.NewServer(exporter.New(s))
}
