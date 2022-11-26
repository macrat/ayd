package ayd_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/lib-ayd"
)

func TestParseProbePluginArgs(t *testing.T) {
	tests := []struct {
		Args   []string
		Target string
		Error  string
	}{
		{
			[]string{"./ayd-test-probe", "foo:bar"},
			"foo:bar",
			"",
		},
		{
			[]string{"./ayd-test-probe"},
			"",
			`invalid argument: should give just 1 argument`,
		},
		{
			[]string{"./ayd-test-probe", "foo:bar", "extra argument"},
			"",
			`invalid argument: should give just 1 argument`,
		},
		{
			[]string{"./ayd-test-probe", "::invalid::"},
			"",
			`invalid target URL: parse "::invalid::": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.Args), func(t *testing.T) {
			args, err := ayd.ParseProbePluginArgsFrom(tt.Args)
			if err != nil {
				if err.Error() != tt.Error {
					t.Errorf("unexpected error: %s", err)
				}
				return
			} else if tt.Error != "" {
				t.Fatalf("expected error but got nil")
			}

			if args.TargetURL.String() != tt.Target {
				t.Errorf("unexpected target URL: %s", args.TargetURL)
			}
		})
	}
}

func TestParseAlertPluginArgs(t *testing.T) {
	tests := []struct {
		Args    []string
		Alert   string
		Time    string
		Status  string
		Latency string
		Target  string
		Message string
		Extra   map[string]interface{}
		Error   string
	}{
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "123.456", "bar:baz", "foo bar", `{"hello":"world"}`},
			"foo:bar",
			"2001-02-03 16:05:06 +0000 UTC",
			"HEALTHY",
			"123.456",
			"bar:baz",
			"foo bar",
			map[string]interface{}{"hello": "world"},
			"",
		},
		{
			[]string{"./ayd-test-alert"},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			`invalid argument: should give exactly 7 arguments`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "123.456", "bar:baz", "foo bar", "{}", "unknown extra arg"},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			`invalid argument: should give exactly 7 arguments`,
		},
		{
			[]string{"./ayd-test-alert", "::invalid::", "2001-02-03T16:05:06Z", "HEALTHY", "123.456", "bar:baz", "foo bar", "{}"},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			`invalid alert URL: parse "::invalid::": missing protocol scheme`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "123.456", "::invalid::", "foo bar", "{}"},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			`invalid target URL: parse "::invalid::": missing protocol scheme`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "this is not a time", "HEALTHY", "123.456", "bar:baz", "foo bar", "{}"},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			`invalid timestamp: invalid format: "this is not a time"`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "not a latency", "bar:baz", "foo bar", "{}"},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			`invalid latency: strconv.ParseFloat: parsing "not a latency": invalid syntax`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "123.456", "bar:baz", "foo bar", `invalid extra`},
			"",
			"",
			"",
			"",
			"",
			"",
			nil,
			"invalid extra values: invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.Args), func(t *testing.T) {
			args, err := ayd.ParseAlertPluginArgsFrom(tt.Args)
			if err != nil {
				if err.Error() != tt.Error {
					t.Errorf("unexpected error: %s", err)
				}
				return
			} else if tt.Error != "" {
				t.Fatalf("expected error but got nil")
			}

			if args.AlertURL.String() != tt.Alert {
				t.Errorf("unexpected alert URL: %s", args.AlertURL)
			}

			if args.Time.String() != tt.Time {
				t.Errorf("unexpected checked time: %s", args.Time)
			}

			if args.Status.String() != tt.Status {
				t.Errorf("unexpected status: %s", args.Status)
			}

			if args.TargetURL.String() != tt.Target {
				t.Errorf("unexpected target URL: %s", args.TargetURL)
			}

			if args.Message != tt.Message {
				t.Errorf("unexpected message: %s", args.Message)
			}

			if diff := cmp.Diff(args.Extra, tt.Extra); diff != "" {
				t.Errorf("unexpected extra\n--- expected ---\n%s--- actual ---\n %s", tt.Extra, args.Extra)
			}
		})
	}
}
