package store

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func Test_fileScanner_longPeriod(t *testing.T) {
	t.Parallel()

	s, err := newFileScanner("testdata/long-period.log", time.Date(2021, 1, 2, 15, 0, 0, 0, time.UTC), time.Date(2021, 1, 2, 17, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to open scanner: %s", err)
	}
	defer s.Close()

	var records []string
	for s.Scan() {
		records = append(records, s.Record().Message)
	}

	expected := []string{"hello world 1", "hello world 2"}
	if !reflect.DeepEqual(expected, records) {
		t.Errorf("unexpected records\nexpected: %v\n but got: %v", expected, records)
	}
}

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

func Benchmark_fileScanner(b *testing.B) {
	p := filepath.Join(b.TempDir(), "test.log")

	baseTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	recordsNum := 1000
	logSize := 0

	f, err := os.Create(p)
	if err != nil {
		b.Fatalf("failed to prepare test log: %s", err)
	}
	for i := 0; i < recordsNum; i++ {
		r := api.Record{
			Time:    baseTime.Add(time.Duration(i) * time.Minute),
			Message: fmt.Sprintf("record %d", i),
		}
		if n, err := f.Write([]byte(r.String() + "\n")); err != nil {
			b.Fatalf("failed to prepare test log: %s", err)
		} else {
			logSize += n
		}
	}
	f.Close()

	b.SetBytes(int64(logSize))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := newFileScanner(p, time.Unix(0, 0), time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			b.Fatalf("failed to open scanner: %s", err)
		}

		for s.Scan() {
			s.Record()
		}
	}
}
