package logconv_test

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/logconv"
	"github.com/macrat/ayd/internal/testutil"
)

func TestToXlsx(t *testing.T) {
	s := testutil.NewStoreWithLog(t)

	r, err := s.OpenLog(time.Unix(0, 0), time.Date(9999, 0, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to open log: %s", err)
	}

	var w bytes.Buffer

	err = logconv.ToXlsx(&w, r, time.Date(2001, 2, 3, 15, 4, 5, 6, time.UTC))
	if err != nil {
		t.Fatalf("failed to convert: %s", err)
	}

	output, err := zip.NewReader(bytes.NewReader(w.Bytes()), int64(w.Len()))
	if err != nil {
		t.Fatalf("failed to open generated file: %s", err)
	}

	var outputLen int
	err = fs.WalkDir(output, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		outputLen++
		return nil
	})
	if err != nil {
		t.Errorf("failed to walk on the output: %s", err)
	}

	snapshot, err := zip.OpenReader("testdata/log.xlsx")
	if err != nil {
		t.Errorf("failed to open snapshot file: %s", err)
	}
	defer snapshot.Close()

	var snapshotLen int
	err = fs.WalkDir(snapshot, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		snapshotLen++

		of, err := output.Open(path)
		if err != nil {
			t.Errorf("failed to open %q from the output: %s", path, err)
			return nil
		}
		defer of.Close()

		o, err := io.ReadAll(of)
		if err != nil {
			t.Errorf("failed to read %q from the output", path)
			return nil
		}

		sf, err := snapshot.Open(path)
		if err != nil {
			t.Errorf("failed to open %q from the snapshot: %s", path, err)
			return nil
		}
		defer sf.Close()

		s, err := io.ReadAll(sf)
		if err != nil {
			t.Errorf("failed to read %q from the snapshot", path)
			return nil
		}

		if !reflect.DeepEqual(o, s) {
			t.Errorf("%q was different", path)
			return nil
		}

		return nil
	})
	if err != nil {
		t.Errorf("failed to walk on the snapshot: %s", err)
	}

	if snapshotLen != outputLen {
		t.Errorf("different number of files found in the output")
	}

	if t.Failed() {
		err = os.MkdirAll("testdata/actual", 0755)
		if err != nil {
			t.Errorf("failed to make testdata/actual: %s", err)
		}

		err := os.WriteFile("testdata/actual/log.xlsx", w.Bytes(), 0644)
		if err != nil {
			t.Errorf("failed to write actual file: %s", err)
		}

		t.Log("unexpected output. please see testdata/actual/log.xlsx")
	}
}
