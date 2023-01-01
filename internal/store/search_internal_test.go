package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func Test_searchLog(t *testing.T) {
	t.Parallel()

	type Test struct {
		target time.Time
		pos    int64
	}
	tests := []struct {
		fname string
		tests []Test
	}{
		{
			"../testutil/testdata/test.log",
			[]Test{
				{time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), 0},
				{time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC), 980},
				{time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC), 0},
				{time.Date(2021, 1, 2, 15, 4, 6, 0, time.UTC), 261},
				{time.Date(2021, 1, 2, 15, 4, 7, 0, time.UTC), 538},
				{time.Date(2021, 1, 2, 15, 4, 8, 0, time.UTC), 669},
				{time.Date(2021, 1, 2, 15, 4, 9, 0, time.UTC), 817},
			},
		},
		{
			"./testdata/long-line.log",
			[]Test{
				{time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), 0},
				{time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC), 514},
				{time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC), 1028},
				{time.Date(2022, 1, 1, 3, 0, 0, 0, time.UTC), 1542},
				{time.Date(2022, 1, 1, 4, 0, 0, 0, time.UTC), 2056},
				{time.Date(2022, 1, 1, 5, 0, 0, 0, time.UTC), 2570},
				{time.Date(2022, 1, 1, 6, 0, 0, 0, time.UTC), 3596},
				{time.Date(2022, 1, 1, 7, 0, 0, 0, time.UTC), 4622},
			},
		},
		{
			"./testdata/empty-line.log",
			[]Test{
				{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), 0},
				{time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC), 111},
				{time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), 229},
				{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), 360},
			},
		},
	}

	for _, tt := range tests {
		fname := tt.fname
		tests := tt.tests

		t.Run(fname, func(t *testing.T) {
			f, err := os.Open(fname)
			if err != nil {
				t.Fatalf("failed to open log: %s", err)
			}
			defer f.Close()

			for _, tt := range tests {
				err := searchLog(f, tt.target, 0)
				if err != nil {
					t.Errorf("%s: unexpected error: %s", tt.target, err)
					continue
				}

				pos, err := f.Seek(0, os.SEEK_CUR)
				if err != nil {
					t.Errorf("%s: failed to current position: %s", tt.target, err)
					continue
				}
				if pos != tt.pos {
					t.Errorf("%s: unexpected position: want=%d actual=%d", tt.target, tt.pos, pos)
				}
			}
		})
	}
}

func Benchmark_searchLog(b *testing.B) {
	p := filepath.Join(b.TempDir(), "test.log")

	baseTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	recordsNum := 10000
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

	f, err = os.Open(p)
	if err != nil {
		b.Fatalf("failed to open test log: %s", err)
	}
	defer f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searchLog(f, baseTime.Add(time.Duration(i*1234%recordsNum)*time.Minute), 1024)
	}
}
