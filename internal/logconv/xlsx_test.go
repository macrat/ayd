package logconv_test

import (
	"bytes"
	"os"
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

	want, err := os.Open("testdata/log.xlsx")
	if err != nil {
		t.Errorf("failed to open snapshot file: %s", err)
	}
	actual := bytes.NewReader(w.Bytes())

	ws, _ := want.Stat()

	if !testutil.XlsxEqual(actual, int64(w.Len()), want, int64(ws.Size())) {
		err = os.MkdirAll("testdata/actual", 0755)
		if err != nil {
			t.Errorf("failed to make testdata/actual: %s", err)
		}

		err := os.WriteFile("testdata/actual/log.xlsx", w.Bytes(), 0644)
		if err != nil {
			t.Errorf("failed to write actual file: %s", err)
		}

		t.Errorf("unexpected output. please check testdata/actual/log.xlsx")
	}
}
