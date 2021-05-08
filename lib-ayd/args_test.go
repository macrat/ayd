package ayd_test

import (
	"fmt"
	"testing"

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
			`invalid argument: invalid target URL: parse "::invalid::": missing protocol scheme`,
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
		Args      []string
		Alert     string
		CheckedAt string
		Status    string
		Target    string
		Message   string
		Error     string
	}{
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "bar:baz", "foo bar"},
			"foo:bar",
			"2001-02-03 16:05:06 +0000 UTC",
			"HEALTHY",
			"bar:baz",
			"foo bar",
			"",
		},
		{
			[]string{"./ayd-test-alert"},
			"",
			"",
			"",
			"",
			"",
			`invalid argument: should give just 5 argument`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "bar:baz", "foo bar", "extra arg"},
			"",
			"",
			"",
			"",
			"",
			`invalid argument: should give just 5 argument`,
		},
		{
			[]string{"./ayd-test-alert", "::invalid::", "2001-02-03T16:05:06Z", "HEALTHY", "bar:baz", "foo bar"},
			"",
			"",
			"",
			"",
			"",
			`invalid argument: invalid alert URL: parse "::invalid::": missing protocol scheme`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "2001-02-03T16:05:06Z", "HEALTHY", "::invalid::", "foo bar"},
			"",
			"",
			"",
			"",
			"",
			`invalid argument: invalid target URL: parse "::invalid::": missing protocol scheme`,
		},
		{
			[]string{"./ayd-test-alert", "foo:bar", "this is not a time", "HEALTHY", "bar:baz", "foo bar"},
			"",
			"",
			"",
			"",
			"",
			`invalid argument: invalid checked time: parsing time "this is not a time" as "2006-01-02T15:04:05Z07:00": cannot parse "this is not a time" as "2006"`,
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

			if args.CheckedAt.String() != tt.CheckedAt {
				t.Errorf("unexpected checked time: %s", args.CheckedAt)
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
		})
	}
}
