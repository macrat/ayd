package testutil

import (
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

func NewStoreWithConsole(t testing.TB, w io.Writer) *store.Store {
	t.Helper()

	s, err := store.New(filepath.Join(t.TempDir(), "ayd.log"), w)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}

	return s
}

func NewStore(t testing.TB) *store.Store {
	t.Helper()

	return NewStoreWithConsole(t, io.Discard)
}

func NewStoreWithLog(t testing.TB) *store.Store {
	fpath := filepath.Join(t.TempDir(), "ayd.log")

	if err := os.WriteFile(fpath, []byte(DummyLog), 0644); err != nil {
		t.Fatalf("failed to prepare test log file: %s", err)
	}

	s, err := store.New(fpath, io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore store: %s", err)
	}

	noRecordTarget := &api.URL{Scheme: "dummy", Fragment: "no-record-yet"}
	s.ActivateTarget(noRecordTarget, noRecordTarget)

	return s
}
