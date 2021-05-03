package main_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
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
	os.Setenv("PATH", origPath+string(filepath.ListSeparator)+filepath.Join(cwd, "testdata"))
	t.Cleanup(func() {
		os.Setenv("PATH", origPath)
	})

	tests := []struct {
		Target  string
		Message string
		Error   string
	}{
		{"exec:ayd-foo-alert", "   ", ""},
		{"exec:ayd-bar-probe", "arg \nenv ayd_target=dummy:failure ayd_status=FAILURE ayd_checked_at=2001-02-03T16:05:06Z", ""},
		{"foo:", "foo: dummy:failure FAILURE 2001-02-03T16:05:06Z", ""},
		{"bar:", "", "unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			alert, err := main.NewAlert(tt.Target)
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
				CausedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
			}

			r := &testutil.DummyReporter{}

			alert.Trigger(ctx, i, r)

			if len(r.Records) != 1 {
				t.Fatalf("unexpected number of records: %d: %#v", len(r.Records), r.Records)
			}

			if r.Records[0].Target.String() != "alert:"+tt.Target {
				t.Errorf("unexpected target URI: %s", r.Records[0].Target)
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
