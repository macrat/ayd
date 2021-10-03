package ayd_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/lib-ayd"
)

func TestRecord(t *testing.T) {
	tokyo := time.FixedZone("UTC+9", +9*60*60)

	tests := []struct {
		String string
		Record ayd.Record
		Error  string
	}{
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\tping:example.com\thello world",
			Record: ayd.Record{
				CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:    &url.URL{Scheme: "ping", Opaque: "example.com"},
				Status:    ayd.StatusHealthy,
				Message:   "hello world",
				Latency:   123456 * time.Microsecond,
			},
		},
		{
			String: "2021-01-02T15:04:05+09:00\tFAILURE\t123.456\texec:/path/to/file.sh\thello world",
			Record: ayd.Record{
				CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:    &url.URL{Scheme: "exec", Opaque: "/path/to/file.sh"},
				Status:    ayd.StatusFailure,
				Message:   "hello world",
				Latency:   123456 * time.Microsecond,
			},
		},
		{
			String: "2021-01-02T15:04:05+09:00\tABORTED\t1234.567\tdummy:#hello\thello world",
			Record: ayd.Record{
				CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:    &url.URL{Scheme: "dummy", Fragment: "hello"},
				Status:    ayd.StatusAborted,
				Message:   "hello world",
				Latency:   1234567 * time.Microsecond,
			},
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456",
			Error:  "invalid record: unexpected column count",
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123abc\tping:example.com\thello world",
			Error:  `invalid format latency: strconv.ParseFloat: parsing "123abc": invalid syntax`,
		},
		{
			String: "2021/01/02 15:04:05\tHEALTHY\t123.456\tping:example.com\thello world",
			Error:  `invalid format checked-at: parsing time "2021/01/02 15:04:05" as "2006-01-02T15:04:05Z07:00": cannot parse "/01/02 15:04:05" as "-"`,
		},
		{
			String: "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\t::invalid target::\thello world",
			Error:  `invalid format target URL: parse "::invalid target::": missing protocol scheme`,
		},
	}

	for _, tt := range tests {
		r, err := ayd.ParseRecord(tt.String)
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

func TestRecord_redact(t *testing.T) {
	record := ayd.Record{
		CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Target:    &url.URL{Scheme: "http", Path: "/path/to/file", User: url.UserPassword("MyName", "HideMe")},
		Status:    ayd.StatusHealthy,
		Message:   "hello world",
		Latency:   123456 * time.Microsecond,
	}

	str := record.String()
	if !strings.Contains(str, "/path/to/file") {
		t.Errorf("record does not contain URL path\n%#v", str)
	}
	if !strings.Contains(str, "MyName") {
		t.Errorf("record does not contain username\n%#v", str)
	}
	if strings.Contains(str, "HideMe") {
		t.Errorf("record contains password\n%#v", str)
	}
}

func TestRecord_json(t *testing.T) {
	t.Run("marshal-and-unmarshal", func(t *testing.T) {
		r1 := ayd.Record{
			CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
			Status:    ayd.StatusHealthy,
			Latency:   123456 * time.Microsecond,
			Target:    &url.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
			Message:   "this is test",
		}

		j, err := json.Marshal(r1)
		if err != nil {
			t.Fatalf("failed to marshal: %s", err)
		}

		var r2 ayd.Record
		err = json.Unmarshal(j, &r2)
		if err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if r1.String() != r2.String() {
			t.Fatalf("source and output is not same\nsource: %s\noutput: %s", r1, r2)
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		source := `{"checked_at":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"dummy:healthy#hello-world", "message":"this is test"}`
		expect := "2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\tdummy:healthy#hello-world\tthis is test"

		var r ayd.Record
		if err := json.Unmarshal([]byte(source), &r); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if r.String() != expect {
			t.Fatalf("unexpected unmarshalled result\nexpected: %s\n but got: %s", expect, r)
		}
	})
}
