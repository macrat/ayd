package testutil

import (
	"io"
	"os"
	"testing"

	"github.com/macrat/ayd/store"
)

func NewStore(t testing.TB) *store.Store {
	t.Helper()

	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	fname := f.Name()
	t.Cleanup(func() {
		os.Remove(fname)
	})
	f.Close()

	s, err := store.New(fname)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	s.Console = io.Discard
	return s
}
