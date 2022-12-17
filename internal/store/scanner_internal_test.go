package store

import (
	"reflect"
	"testing"
	"time"
)

func Test_fileScannerSet(t *testing.T) {
	t.Parallel()

	s, err := newFileScannerSet([]string{"testdata/dummy-a.log", "testdata/no-such-file.log", "testdata/dummy-b.log", "testdata/invalid.log"}, time.Unix(0, 0), time.Unix(1<<60-1, 0))
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

	if s.Scan() {
		t.Errorf("scanner returns a record after scan all records:\n%s", s.Record())
	}
}

func Test_fileScannerSet_empty(t *testing.T) {
	t.Parallel()

	s, err := newFileScannerSet([]string{"testdata/no-such-file.log"}, time.Unix(0, 0), time.Unix(1<<60-1, 0))
	if err != nil {
		t.Fatalf("failed to open scanner set: %s", err)
	}
	defer s.Close()

	if len(s.scanners) != 0 {
		t.Errorf("unexpected number of record files found:\n%#v", s)
	}

	if s.Scan() {
		t.Fatalf("expected empty scanner but got record:\n%s", s.Record())
	}
}