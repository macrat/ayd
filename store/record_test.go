package store_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/macrat/ayd/store"
)

func TestRecord(t *testing.T) {
	tokyo := time.FixedZone("UTC+9", +9*60*60)

	tests := []struct {
		String string
		Record store.Record
		Error  string
	}{
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\tping:example.com\thello world",
			Record: store.Record{
				CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:    &url.URL{Scheme: "ping", Opaque: "example.com"},
				Status:    store.STATUS_HEALTHY,
				Message:   "hello world",
				Latency:   123456 * time.Microsecond,
			},
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\texec:/path/to/file.sh\thello world",
			Record: store.Record{
				CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:    &url.URL{Scheme: "exec", Opaque: "/path/to/file.sh"},
				Status:    store.STATUS_HEALTHY,
				Message:   "hello world",
				Latency:   123456 * time.Microsecond,
			},
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456",
			Error:  "unexpected column count",
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123abc\tping:example.com\thello world",
			Error:  `strconv.ParseFloat: parsing "123abc": invalid syntax`,
		},
		{
			String: "2021/01/02 15:04:05\tHEALTHY\t123.456\tping:example.com\thello world",
			Error:  `parsing time "2021/01/02 15:04:05" as "2006-01-02T15:04:05Z07:00": cannot parse "/01/02 15:04:05" as "-"`,
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\t::invalid target::\thello world",
			Error:  `parse "::invalid target::": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		r, err := store.ParseRecord(tt.String)
		if tt.Error != "" {
			if err == nil || tt.Error != err.Error() {
				t.Errorf("expected error when parse %#v\nexpected \"%s\" but got \"%s\"", tt.String, tt.Error, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("failed to parse %#v: %s", tt.String, err)
			continue
		}

		if !r.CheckedAt.Equal(tt.Record.CheckedAt) {
			t.Errorf("unexpected parsed timestamp\nexpected: %#v\n but got: %#v", tt.Record.CheckedAt, r.CheckedAt)
		}

		if tt.Record.Target.String() != r.Target.String() {
			t.Errorf("unexpected parsed target\nexpected: %s\n but got: %s", tt.Record.Target, r.Target)
		}

		if tt.Record.Status != r.Status {
			t.Errorf("unexpected parsed status\nexpected: %s\n but got: %s", tt.Record.Status, r.Status)
		}

		if tt.Record.Latency != r.Latency {
			t.Errorf("unexpected parsed latency\nexpected: %#v\n but got: %#v", tt.Record.Latency, r.Latency)
		}

		if tt.Record.Message != r.Message {
			t.Errorf("unexpected parsed message\nexpected: %#v\n but got: %#v", tt.Record.Message, r.Message)
		}

		if tt.Record.String() != tt.String {
			t.Errorf("expected: %#v\n but got: %#v", tt.String, tt.Record.String())
		}
	}
}

func TestUnescapeMessage(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{`hello world`, `hello world`},
		{`"hello"world`, `"hello"world`},
		{`hello\tworld`, "hello\tworld"},
		{`hello\nworld`, "hello\nworld"},
		{`hello\r\nworld`, "hello\\r\nworld"},
		{`\\hello\\world\\\\\n`, "\\hello\\world\\\\\n"},
		{`\n`, "\n"},
		{``, ""},
	}

	for _, tt := range tests {
		got := store.UnescapeMessage(tt.Input)
		if got != tt.Output {
			t.Errorf("%#v: unexpected result\nexpected: %#v\n but got: %#v", tt.Input, tt.Output, got)
		}
	}
}

func TestEscapeMessage(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{`hello world`, `hello world`},
		{`"hello"world`, `"hello"world`},
		{"hello\tworld", `hello\tworld`},
		{"\thello\tworld\t", `\thello\tworld\t`},
		{"\n\nhello\nworld\n", `\n\nhello\nworld\n`},
		{`\n\t\\`, `\\n\\t\\\\`},
		{"\n", `\n`},
		{"", ``},
	}

	for _, tt := range tests {
		got := store.EscapeMessage(tt.Input)
		if got != tt.Output {
			t.Errorf("%#v: unexpected result\nexpected: %#v\n but got: %#v", tt.Input, tt.Output, got)
		}
	}
}
