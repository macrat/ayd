package logconv_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func Assert(t *testing.T, name string, actual []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read %s", path)
		return
	}

	if reflect.DeepEqual(want, actual) {
		return
	}

	err = os.MkdirAll("testdata/actual", 0755)
	if err != nil {
		t.Errorf("failed to make testdata/actual: %s", err)
	}

	path = filepath.Join("testdata/actual", name)
	err = os.WriteFile(path, actual, 0644)
	if err != nil {
		t.Errorf("failed to write to %s: %s", err, path)
	}

	t.Errorf("unexpected output. please see %s", path)
}
