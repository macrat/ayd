package testutil

import (
	"archive/zip"
	"io"
	"io/fs"
	"reflect"
)

func XlsxEqual(actual io.ReaderAt, actualSize int64, want io.ReaderAt, wantSize int64) (same bool) {
	output, err := zip.NewReader(actual, actualSize)
	if err != nil {
		return false
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
		return false
	}

	snapshot, err := zip.NewReader(want, wantSize)
	if err != nil {
		return false
	}

	defer func() {
		if recover() != nil {
			same = false
		}
	}()

	var snapshotLen int
	err = fs.WalkDir(snapshot, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		snapshotLen++

		of, err := output.Open(path)
		if err != nil {
			return nil
		}
		defer of.Close()

		o, err := io.ReadAll(of)
		if err != nil {
			panic(err)
		}

		sf, err := snapshot.Open(path)
		if err != nil {
			return nil
		}
		defer sf.Close()

		s, err := io.ReadAll(sf)
		if err != nil {
			panic(err)
		}

		if !reflect.DeepEqual(o, s) {
			panic(err)
		}

		return nil
	})
	if err != nil {
		return false
	}

	return snapshotLen == outputLen
}
