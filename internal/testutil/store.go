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

type StoreOption func(*storeOptionStruct)

type storeOptionStruct struct {
	InstanceName string
	Console      io.Writer
	WithLog      bool
}

func WithInstanceName(name string) StoreOption {
	return func(o *storeOptionStruct) {
		o.InstanceName = name
	}
}

func WithConsole(w io.Writer) StoreOption {
	return func(o *storeOptionStruct) {
		o.Console = w
	}
}

func WithLog() StoreOption {
	return func(o *storeOptionStruct) {
		o.WithLog = true
	}
}

func NewStore(t testing.TB, opts ...StoreOption) *store.Store {
	t.Helper()

	var opt storeOptionStruct
	for _, o := range opts {
		o(&opt)
	}

	fpath := filepath.Join(t.TempDir(), "ayd.log")

	if opt.WithLog {
		if err := os.WriteFile(fpath, []byte(DummyLog), 0644); err != nil {
			t.Fatalf("failed to prepare test log file: %s", err)
		}
	}

	name := opt.InstanceName

	console := io.Discard
	if opt.Console != nil {
		console = opt.Console
	}

	s, err := store.New(name, fpath, console)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}

	if opt.WithLog {
		if err = s.Restore(); err != nil {
			t.Fatalf("failed to restore store: %s", err)
		}

		noRecordTarget := &api.URL{Scheme: "dummy", Fragment: "no-record-yet"}
		s.ActivateTarget(noRecordTarget, noRecordTarget)
	}

	return s
}
