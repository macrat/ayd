package logconv_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/logconv"
	"github.com/macrat/ayd/internal/testutil"
)

func TestToLTSV(t *testing.T) {
	s := testutil.NewStoreWithLog(t)

	r, err := s.OpenLog(time.Unix(0, 0), time.Date(9999, 0, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to open log: %s", err)
	}

	var w bytes.Buffer

	err = logconv.ToLTSV(&w, r)
	if err != nil {
		t.Fatalf("failed to convert: %s", err)
	}
	Assert(t, "log.ltsv", w.Bytes())
}
