package scheme_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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

	type Test struct {
		TargetIn  string
		TargetOut string
		Message   string
		Error     string
	}
	tests := []Test{
		{"exec:ayd-foo-alert", "alert:exec:ayd-foo-alert", `{"time":"2001-02-03T16:05:06Z","status":"HEALTHY","latency":123.456,"target":"","message":""}` + "\n---\nexit_code: 0", ""},
		{"exec:ayd-bar-probe", "alert:exec:ayd-bar-probe", "arg \"\"\nenv ayd_time=2001-02-03T16:05:06Z ayd_status=FAILURE ayd_latency=12.345 ayd_target=dummy:failure ayd_message=foobar ayd_extra={\"hello\":\"world\"}\n---\nexit_code: 0", ""},
		{"bar:", "", "", "unsupported scheme"},

		{"ftp://example.com", "", "", "unsupported scheme for alert"},
		{"ftps://example.com", "", "", "unsupported scheme for alert"},
		{"ping://example.com", "", "", "unsupported scheme for alert"},
		{"ping4://example.com", "", "", "unsupported scheme for alert"},
		{"ping6://example.com", "", "", "unsupported scheme for alert"},
		{"tcp://example.com", "", "", "unsupported scheme for alert"},
		{"tcp4://example.com", "", "", "unsupported scheme for alert"},
		{"tcp6://example.com", "", "", "unsupported scheme for alert"},
		{"dns://example.com", "", "", "unsupported scheme for alert"},
		{"dns4://example.com", "", "", "unsupported scheme for alert"},
		{"dns6://example.com", "", "", "unsupported scheme for alert"},

		{"::", "", "", "invalid URL"},
		{"ayd:test:internal-url", "", "", "unsupported scheme"},
		{"alert:", "", "", "unsupported scheme"},
		{"alert-abc:", "", "", "unsupported scheme"},
		{"of-course-no-such-plugin:", "", "", "unsupported scheme"},
	}

	if runtime.GOOS != "windows" {
		// Windows can not run this test because bat doesn't support double quote character in argument :(
		tests = append(
			tests,
			Test{"foo-bar:hello-world", "alert:foo-bar:hello-world", "foo-bar:hello-world\n---\n" + `record: {"hello":"world","latency":12.345,"message":"foobar","status":"FAILURE","target":"dummy:failure","time":"2001-02-03T16:05:06Z"}`, ""},
			Test{"foo:", "alert:foo:", "foo:\n---\n" + `record: {"hello":"world","latency":12.345,"message":"foobar","status":"FAILURE","target":"dummy:failure","time":"2001-02-03T16:05:06Z"}`, ""},
			Test{"foo:hello-world", "alert:foo:hello-world", "foo:hello-world\n---\n" + `record: {"hello":"world","latency":12.345,"message":"foobar","status":"FAILURE","target":"dummy:failure","time":"2001-02-03T16:05:06Z"}`, ""},
		)
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
				Time:    time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Status:  api.StatusFailure,
				Latency: 12345 * time.Microsecond,
				Target:  &api.URL{Scheme: "dummy", Opaque: "failure"},
				Message: "foobar",
				Extra: map[string]interface{}{
					"hello": "world",
				},
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

			if r.Records[0].ReadableMessage() != tt.Message {
				t.Errorf("--- expected message ---\n%s\n--- actual message ---\n%s", tt.Message, r.Records[0].ReadableMessage())
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
			Target:  &api.URL{Scheme: "dummy", Opaque: "failure"},
			Status:  api.StatusFailure,
			Time:    time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
			Message: "foobar",
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
		if r.Records[0].Message != `invalid record: invalid character 'h' in literal true (expecting 'r'): "this is invalid record"` {
			t.Errorf("unexpected message of record: %s", r.Records[0].Message)
		}
	})
}

func TestAlertReporter(t *testing.T) {
	r1 := &testutil.DummyReporter{}
	r2 := scheme.AlertReporter{&api.URL{Scheme: "alert", Opaque: "dummy:"}, r1}

	r2.Report(&api.URL{Scheme: "dummy"}, api.Record{
		Target:  &api.URL{Scheme: "dummy", Opaque: "another"},
		Message: "test-message",
	})
	r2.Report(&api.URL{Scheme: "dummy"}, api.Record{
		Target:  &api.URL{Scheme: "ayd", Opaque: "test:internal-log"},
		Message: "something log",
	})

	if len(r1.Records) != 2 {
		t.Fatalf("unexpected number of records: %d: %v", len(r1.Records), r1.Records)
	}

	if r1.Records[0].String() != `{"time":"0001-01-01T00:00:00Z", "status":"UNKNOWN", "latency":0.000, "target":"alert:dummy:another", "message":"test-message"}` {
		t.Errorf("unexpected 1st record: %s", r1.Records[0])
	}

	if r1.Records[1].String() != `{"time":"0001-01-01T00:00:00Z", "status":"UNKNOWN", "latency":0.000, "target":"ayd:test:internal-log", "message":"something log"}` {
		t.Errorf("unexpected 2nd record: %s", r1.Records[1])
	}
}

func TestAlerterSet(t *testing.T) {
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
		{"duplicated", []string{"dummy:?message=abc", "dummy:?message=abc", "dummy:?message=def"}, []string{"abc", "def"}, ""},
		{"invalid", []string{"dummy:#its_okay", "::invalid::", "no.such:abc", "dummy:#its_also_okay"}, nil, "invalid alert URL:\n  ::invalid::: invalid URL\n  no.such:abc: unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			as, err := scheme.NewAlerterSet(tt.URLs)
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
				Target:  &api.URL{Scheme: "dummy", Opaque: "failure"},
				Status:  api.StatusFailure,
				Time:    time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Message: "foobar",
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

func TestAlerterSet_blocking(t *testing.T) {
	t.Parallel()

	as, err := scheme.NewAlerterSet([]string{"dummy:?latency=500ms", "dummy:?latency=1000ms"})
	if err != nil {
		t.Fatalf("failed to create a new set: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rec := api.Record{
		Target:  &api.URL{Scheme: "dummy", Opaque: "failure"},
		Status:  api.StatusFailure,
		Time:    time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
		Message: "foobar",
	}

	r := &testutil.DummyReporter{}

	stime := time.Now()
	as.Alert(ctx, r, rec)
	delay := time.Since(stime)

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
				Time:    time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Target:  &api.URL{Scheme: "dummy", Opaque: "failure"},
				Status:  api.StatusFailure,
				Latency: 123456 * time.Microsecond,
				Message: "test-message",
				Extra: map[string]interface{}{
					"hello": "world",
				},
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
			if ok, _ := regexp.MatchString("^"+tt.MessagePattern+"$", r.ReadableMessage()); !ok {
				t.Errorf("unexpected message\n--- expected -----\n%s\n--- actual -----\n%s", tt.MessagePattern, r.ReadableMessage())
			}
		})
	}
}
