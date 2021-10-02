package testutil

import (
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/macrat/ayd/internal/store"
)

func NewStore(t testing.TB) *store.Store {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), "ayd.log"))
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	s.Console = io.Discard
	return s
}

//go:embed testdata/test.log
var rawLog []byte

func NewStoreWithLog(t testing.TB) *store.Store {
	fpath := filepath.Join(t.TempDir(), "ayd.log")

	if err := os.WriteFile(fpath, rawLog, 0644); err != nil {
		t.Fatalf("failed to prepare test log file: %s", err)
	}

	s, err := store.New(fpath)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	s.Console = io.Discard

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore store: %s", err)
	}

	return s
}
