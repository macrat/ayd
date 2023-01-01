//go:build linux || darwin
// +build linux darwin

package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Test_fileScannerSet_permissionDenied(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "forbidden.log")

	if err := os.Mkdir(path, 0); err != nil {
		t.Fatalf("failed to make test directory: %s", err)
	}

	s, err := newFileScannerSet([]string{"testdata/dummy-a.log", path, "testdata/dummy-b.log"}, time.Unix(0, 0), time.Unix(1<<60-1, 0))
	if err == nil {
		s.Close()
		t.Fatalf("expected error but got nil")
	} else if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unexpected error: %s", err)
	}
}
