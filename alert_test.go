package main_test

import (
	"context"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestAlert(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current path: %s", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", origPath+":"+path.Join(cwd, "testdata"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})

	tests := []struct {
		Target   string
		External string
		Message  string
		Error    string
	}{
		{"exec:./testdata/ayd-foo-alert", "http://localhost:8000", "foo FAILURE dummy:failure", ""},
		{"exec:./testdata/ayd-bar-probe", "http://localhost:8000", "bar FAILURE dummy:failure", ""},
		{"foo:", "http://localhost:8000", "foo FAILURE dummy:failure", ""},
		{"bar:", "http://localhost:8000", "", "unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			alert, err := main.NewAlert(tt.Target, tt.External)
			if err != nil && err.Error() != tt.Error {
				t.Fatalf("unexpected error: %s", err)
			} else if tt.Error != "" {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			i := &store.Incident{
				Target:   &url.URL{Scheme: "dummy", Opaque: "failure"},
				Status:   store.STATUS_FAILURE,
				CausedAt: time.Now(),
			}

			r := &testutil.DummyReporter{}

			alert.Trigger(ctx, i, r)

			if len(r.Records) != 1 {
				t.Fatalf("unexpected number of records: %d: %#v", len(r.Records), r.Records)
			}

			if r.Records[0].Status != store.STATUS_HEALTHY {
				t.Errorf("failed to trigger alert: %s", r.Records[0].Status)
			}

			if r.Records[0].Message != tt.Message {
				t.Errorf("unexpected message of record: %s", r.Records[0].Message)
			}
		})
	}
}
