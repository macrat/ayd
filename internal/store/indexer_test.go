package store

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_indexer(t *testing.T) {
	idx := newIndexer()
	idx.interval = 3

	if err := idx.AppendEntry(0, 100, 1); err != nil {
		t.Fatalf("failed to append log entry: %s", err)
	}

	if err := idx.AppendEntry(0, 50, 2); err != ErrLogUnmatch {
		t.Fatalf("append log entry should return ErrLogUnmatch but got: %s", err)
	}

	for i := int64(2); i <= 4; i++ {
		if err := idx.AppendEntry(i*50, i*50+50, i); err != nil {
			t.Fatalf("failed to append log entry[%d]: %s", i, err)
		}
	}

	if err := idx.AppendEntry(250, 300, 10); err != nil {
		t.Fatalf("failed to append log entry: %s", err)
	}

	for i := int64(6); i <= 7; i++ {
		if err := idx.AppendEntry(i*50, i*50+50, i); err != nil {
			t.Fatalf("failed to append log entry[%d]: %s", i, err)
		}
	}

	idx.Lock()
	if err := idx.AppendInvalidRangeWithoutLock(400, 500); err != nil {
		t.Fatalf("failed to append log entry: %s", err)
	}
	idx.Unlock()

	if err := idx.AppendEntry(500, 550, 2); err != nil {
		t.Fatalf("failed to append log entry: %s", err)
	}

	// So far, the entries looks like this.
	//
	//   time  start  end  period
	//   1     0      100  0
	//   2     100    150  0
	//   3     150    200  0
	//   4     200    250  1
	//   10    250    300  1
	//   6     300    350  1
	//   7     350    400  2
	//   -     400    500  -
	//   2     500    550  2

	tests := []struct {
		Since int64
		Until int64
		Want  []logRange
	}{
		{0, 0, nil},
		{0, 1, []logRange{{0, 200, 3}}},
		{2, 3, []logRange{{0, 200, 3}, {500, 550, 1}}},
		{1, 4, []logRange{{0, 350, 6}, {500, 550, 1}}},
		{5, 8, []logRange{{200, 400, 4}}},
		{10, 20, []logRange{{200, 350, 3}}},
		{11, 20, nil},
	}

	for _, tt := range tests {
		if diff := cmp.Diff(tt.Want, idx.Search(tt.Since, tt.Until)); diff != "" {
			t.Errorf("unexpected range: interested period is %d-%d\n%s", tt.Since, tt.Until, diff)
		}
	}
}
