package scheme_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestAlerter(t *testing.T) {
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
			alert, err := scheme.NewAlerter(tt.TargetIn)
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

			alert.Alert(ctx, r, rec)

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
				t.Errorf("--- expected message ---\n%s\n--- actual message ---\n%s", tt.Message, r.Records[0].Message)
			}
		})
	}

	t.Run("broken:", func(t *testing.T) {
		alert, err := scheme.NewAlerter("broken:")
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

		alert.Alert(ctx, r, rec)

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

func TestAlertReporter(t *testing.T) {
	r1 := &testutil.DummyReporter{}
	r2 := scheme.AlertReporter{&url.URL{Scheme: "alert", Opaque: "dummy:"}, r1}

	r2.Report(&url.URL{Scheme: "dummy"}, api.Record{
		Target:  &url.URL{Scheme: "dummy", Opaque: "another"},
		Message: "test-message",
	})
	r2.Report(&url.URL{Scheme: "dummy"}, api.Record{
		Target:  &url.URL{Scheme: "ayd", Opaque: "test:internal-log"},
		Message: "something log",
	})

	if len(r1.Records) != 2 {
		t.Fatalf("unexpected number of records: %d: %v", len(r1.Records), r1.Records)
	}

	if r1.Records[0].String() != "0001-01-01T00:00:00Z	UNKNOWN	0.000	alert:dummy:another	test-message" {
		t.Errorf("unexpected 1st record: %s", r1.Records[0])
	}

	if r1.Records[1].String() != "0001-01-01T00:00:00Z	UNKNOWN	0.000	ayd:test:internal-log	something log" {
		t.Errorf("unexpected 2nd record: %s", r1.Records[1])
	}
}

func TestAlertSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name     string
		URLs     []string
		Messages []string
		Error    string
	}{
		{"empty", []string{}, []string{}, ""},
		{"single", []string{"dummy:?message=abc"}, []string{"abc"}, ""},
		{"multiple", []string{"dummy:?message=abc", "dummy:?message=def"}, []string{"abc", "def"}, ""},
		{"invalid", []string{"dummy:#its_okay", "::invalid::", "no.such:abc", "dummy:#its_also_okay"}, nil, "invalid alert URL:\n  ::invalid::: invalid URL\n  no.such:abc: unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			as, err := scheme.NewAlertSet(tt.URLs)
			if tt.Error != "" {
				if err == nil {
					t.Fatalf("expected error but returns nil")
				}
				if err.Error() != tt.Error {
					t.Errorf("unexpected error\n--- expected ---\n%s\n--- but got ---\n%s", tt.Error, err)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error\n%s", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			rec := api.Record{
				Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
				Status:    api.StatusFailure,
				CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Message:   "foobar",
			}

			r := &testutil.DummyReporter{}

			as.Alert(ctx, r, rec)

			if len(r.Records) != len(tt.Messages) {
				t.Fatalf("expected %d records but got %d records", len(tt.Messages), len(r.Records))
			}

			for _, expect := range tt.Messages {
				ok := false
				for _, found := range r.Records {
					if found.Message == expect {
						ok = true
						break
					}
				}

				if !ok {
					t.Errorf("expected message %#v was not found", expect)
				}
			}
		})
	}
}

func TestAlertSet_blocking(t *testing.T) {
	t.Parallel()

	as, err := scheme.NewAlertSet([]string{"dummy:?latency=500ms", "dummy:?latency=1000ms"})
	if err != nil {
		t.Fatalf("failed to create a new set: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rec := api.Record{
		Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
		Status:    api.StatusFailure,
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
		Message:   "foobar",
	}

	r := &testutil.DummyReporter{}

	stime := time.Now()
	as.Alert(ctx, r, rec)
	delay := time.Now().Sub(stime)

	if len(r.Records) != 2 {
		t.Errorf("unexpected number of records\n%v", r.Records)
	}

	if delay < 1*time.Second {
		t.Errorf("expected to blocking during alert function running but returns too fast: %s", delay)
	}
}

func AssertAlert(t *testing.T, tests []ProbeTest, timeout int) {
	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			a, err := scheme.NewAlerter(tt.Target)
			if err != nil {
				if ok, _ := regexp.MatchString("^"+tt.ParseErrorPattern+"$", err.Error()); !ok {
					t.Fatalf("unexpected error on create probe: %s", err)
				}
				return
			} else if tt.ParseErrorPattern != "" {
				t.Fatal("expected error on create probe but got nil")
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()

			rs := testutil.RunAlert(ctx, a, api.Record{
				CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
				Status:    api.StatusFailure,
				Latency:   123456 * time.Microsecond,
				Message:   "test-message",
			})

			if len(rs) != 1 {
				t.Fatalf("got unexpected number of results: %d\n%v", len(rs), rs)
			}

			r := rs[0]
			if r.Target.String() != "alert:"+tt.Target {
				t.Errorf("got a record of unexpected target: %s", r.Target)
			}
			if r.Status != tt.Status {
				t.Errorf("expected status is %s but got %s", tt.Status, r.Status)
			}
			if ok, _ := regexp.MatchString("^"+tt.MessagePattern+"$", r.Message); !ok {
				t.Errorf("expected message is match to %#v but got %#v", tt.MessagePattern, r.Message)
			}
		})
	}
}
