package store

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func Test_fileScannerSet(t *testing.T) {
	t.Parallel()

	s, err := newFileScannerSet([]string{"testdata/dummy-a.log", "testdata/dummy-b.log", "testdata/invalid.log"}, time.Unix(0, 0), time.Unix(1<<60-1, 0))
	if err != nil {
		t.Fatalf("failed to open scanner set: %s", err)
	}
	defer s.Close()

	if len(s.scanners) != 2 {
		t.Errorf("unexpected number of record files found:\n%#v", s)
	}

	var actual []string
	for s.Scan() {
		actual = append(actual, s.Record().Message)
	}

	want := []string{"1", "2", "3", "4", "5", "6"}
	if !reflect.DeepEqual(want, actual) {
		t.Errorf("unexpected records found:\nexpected: %#v\n but got: %#v", want, actual)
	}
}

func Test_fileScannerSet_withMissingFile(t *testing.T) {
	t.Parallel()

	s, err := newFileScannerSet([]string{"testdata/dummy-a.log", "testdata/dummy-b.log", "testdata/no-such-file.log"}, time.Unix(0, 0), time.Unix(1<<60-1, 0))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("unexpected error: %s", err)
	}
	if s != nil {
		t.Errorf("result should nil but got %#v", s)
		s.Close()
	}
}
