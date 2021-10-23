package alert_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/alert"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
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
		TargetIn  string
		TargetOut string
		Message   string
		Error     string
	}{
		{"exec:ayd-foo-alert", "alert:exec:ayd-foo-alert", "2001-02-03T16:05:06Z\tHEALTHY\t123.456\t\t\"     \"", ""},
		{"exec:ayd-bar-probe", "alert:exec:ayd-bar-probe", "arg \"\"\nenv ayd_checked_at=2001-02-03T16:05:06Z ayd_status=FAILURE ayd_latency=12.345 ayd_target=dummy:failure ayd_message=foobar", ""},
		{"foo:", "alert:foo:", "\"foo: 2001-02-03T16:05:06Z FAILURE 12.345 dummy:failure foobar\"", ""},
		{"foo:hello-world", "alert:foo:hello-world", "\"foo:hello-world 2001-02-03T16:05:06Z FAILURE 12.345 dummy:failure foobar\"", ""},
		{"foo-bar:hello-world", "alert:foo-bar:hello-world", "\"foo-bar:hello-world 2001-02-03T16:05:06Z FAILURE 12.345 dummy:failure foobar\"", ""},
		{"bar:", "", "", "unsupported scheme"},
		{"::", "", "", "invalid URL"},
		{"ayd:test:internal-url", "", "", "unsupported scheme"},
		{"alert:", "", "", "unsupported scheme"},
		{"alert-abc:", "", "", "unsupported scheme"},
		{"of-course-no-such-plugin:", "", "", "unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.TargetIn, func(t *testing.T) {
			alert, err := alert.New(tt.TargetIn)
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

			rec := api.Record{
				CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Status:    api.StatusFailure,
				Latency:   12345 * time.Microsecond,
				Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
				Message:   "foobar",
			}

			r := &testutil.DummyReporter{}

			alert.Trigger(ctx, rec, r)

			if len(r.Records) != 1 {
				t.Fatalf("unexpected number of records: %d: %v", len(r.Records), r.Records)
			}

			if r.Records[0].Target.String() != tt.TargetOut {
				t.Errorf("unexpected target URL: %s", r.Records[0].Target)
			}

			if r.Records[0].Status != api.StatusHealthy {
				t.Errorf("unexpected status of record: %s", r.Records[0].Status)
			}

			if r.Records[0].Message != tt.Message {
				t.Errorf("unexpected message of record: %s", r.Records[0].Message)
			}
		})
	}

	t.Run("broken:", func(t *testing.T) {
		alert, err := alert.New("broken:")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		rec := api.Record{
			Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
			Status:    api.StatusFailure,
			CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
			Message:   "foobar",
		}

		r := &testutil.DummyReporter{}

		alert.Trigger(ctx, rec, r)

		if len(r.Records) != 2 {
			t.Fatalf("unexpected number of records: %d: %v", len(r.Records), r.Records)
		}

		if r.Records[0].Target.String() != "ayd:alert:plugin:broken:" {
			t.Errorf("unexpected target URL: %s", r.Records[0].Target)
		}
		if r.Records[0].Status != api.StatusUnknown {
			t.Errorf("unexpected status of record: %s", r.Records[0].Status)
		}
		if r.Records[0].Message != `invalid record: unexpected column count: "this is invalid record"` {
			t.Errorf("unexpected message of record: %s", r.Records[0].Message)
		}
	})
}

func TestReplaceReporter(t *testing.T) {
	r1 := &testutil.DummyReporter{}
	r2 := alert.ReplaceReporter{&url.URL{Scheme: "alert", Opaque: "dummy:hello-world"}, r1}

	r2.Report(api.Record{
		Target:  &url.URL{Scheme: "dummy", Opaque: "another"},
		Message: "hello world",
	})
	r2.Report(api.Record{
		Target:  &url.URL{Scheme: "ayd", Opaque: "test:internal-log"},
		Message: "something log",
	})

	if len(r1.Records) != 2 {
		t.Fatalf("unexpected number of records: %d: %v", len(r1.Records), r1.Records)
	}

	if r1.Records[0].String() != "0001-01-01T00:00:00Z	UNKNOWN	0.000	alert:dummy:hello-world	hello world" {
		t.Errorf("unexpected 1st record: %s", r1.Records[0])
	}

	if r1.Records[1].String() != "0001-01-01T00:00:00Z	UNKNOWN	0.000	ayd:test:internal-log	something log" {
		t.Errorf("unexpected 2nd record: %s", r1.Records[1])
	}
}

func TestAlertReporter(t *testing.T) {
	r1 := &testutil.DummyReporter{}
	r2 := alert.AlertReporter{r1}

	r2.Report(api.Record{
		Target:  &url.URL{Scheme: "dummy", Opaque: "another"},
		Message: "hello world",
	})
	r2.Report(api.Record{
		Target:  &url.URL{Scheme: "ayd", Opaque: "test:internal-log"},
		Message: "something log",
	})

	if len(r1.Records) != 2 {
		t.Fatalf("unexpected number of records: %d: %v", len(r1.Records), r1.Records)
	}

	if r1.Records[0].String() != "0001-01-01T00:00:00Z	UNKNOWN	0.000	alert:dummy:another	hello world" {
		t.Errorf("unexpected 1st record: %s", r1.Records[0])
	}

	if r1.Records[1].String() != "0001-01-01T00:00:00Z	UNKNOWN	0.000	ayd:test:internal-log	something log" {
		t.Errorf("unexpected 2nd record: %s", r1.Records[1])
	}
}
