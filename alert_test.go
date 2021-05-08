package main_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/macrat/ayd"
	api "github.com/macrat/ayd/lib-ayd"
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
		{"exec:ayd-foo-alert", "2001-02-03T16:05:06Z\tHEALTHY\t123.456\t\t\"    \"", ""},
		{"exec:ayd-bar-probe", "arg \"\"\nenv ayd_checked_at=2001-02-03T16:05:06Z ayd_status=FAILURE ayd_target=dummy:failure ayd_message=foobar", ""},
		{"foo:", "\"foo: 2001-02-03T16:05:06Z FAILURE dummy:failure foobar\"", ""},
		{"bar:", "", "unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			alert, err := main.NewAlert(tt.Target)
			if err != nil {
				if err.Error() != tt.Error {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			} else if tt.Error != "" {
				t.Fatal("expected error but got nil")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			i := &api.Incident{
				Target:   &url.URL{Scheme: "dummy", Opaque: "failure"},
				Status:   api.StatusFailure,
				CausedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Message:  "foobar",
			}

			r := &testutil.DummyReporter{}

			alert.Trigger(ctx, i, r)

			if len(r.Records) != 1 {
				t.Fatalf("unexpected number of records: %d: %#v", len(r.Records), r.Records)
			}

			if r.Records[0].Target.String() != "alert:"+tt.Target {
				t.Errorf("unexpected target URL: %s", r.Records[0].Target)
			}

			if r.Records[0].Status != api.StatusHealthy {
				t.Errorf("failed to trigger alert: %s", r.Records[0].Status)
			}

			if r.Records[0].Message != tt.Message {
				t.Errorf("unexpected message of record: %s", r.Records[0].Message)
			}
		})
	}
}
