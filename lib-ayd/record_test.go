package ayd_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/lib-ayd"
)

func TestURLToStr(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{"https://examle.com/あ?い=う#え#", "https://examle.com/%E3%81%82?い=う#え%23"},
		{"a:/b", "a:/b"},
		{"a:///b", "a:///b"},
	}

	for _, tt := range tests {
		t.Run(tt.Output, func(t *testing.T) {
			u, err := ayd.ParseURL(tt.Input)
			if err != nil {
				t.Fatalf("faield to prepare test input: %s", err)
			}

			s := u.String()
			if s != tt.Output {
				t.Errorf("expected: %s\n but got: %s", tt.Output, s)
			}
		})
	}
}

func BenchmarkURL_String(b *testing.B) {
	u := &ayd.URL{
		Scheme:   "dummy",
		Opaque:   "healthy",
		Fragment: "hello-world#こんにちは世界",
	}

	for i := 0; i < b.N; i++ {
		_ = u.String()
	}
}

func TestRecord(t *testing.T) {
	tokyo := time.FixedZone("UTC+9", +9*60*60)

	tests := []struct {
		String string
		Encode string
		Record ayd.Record
		Error  string
	}{
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world"}`,
			Record: ayd.Record{
				Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:  &ayd.URL{Scheme: "ping", Opaque: "example.com"},
				Status:  ayd.StatusHealthy,
				Message: "hello world",
				Latency: 123456 * time.Microsecond,
			},
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"FAILURE", "latency":123.456, "target":"exec:/path/to/file.sh", "message":"hello world"}`,
			Record: ayd.Record{
				Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:  &ayd.URL{Scheme: "exec", Opaque: "/path/to/file.sh"},
				Status:  ayd.StatusFailure,
				Message: "hello world",
				Latency: 123456 * time.Microsecond,
			},
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"ABORTED", "latency":1234.567, "target":"dummy:#hello", "message":"hello world"}`,
			Record: ayd.Record{
				Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:  &ayd.URL{Scheme: "dummy", Fragment: "hello"},
				Status:  ayd.StatusAborted,
				Message: "hello world",
				Latency: 1234567 * time.Microsecond,
			},
		},
		{
			String: `{"time":"2021-01-02 15:04:05+09:00", "status":"Degrade", "latency":1027.890, "target":"dummy:"}`,
			Encode: `{"time":"2021-01-02T15:04:05+09:00", "status":"DEGRADE", "latency":1027.890, "target":"dummy:"}`,
			Record: ayd.Record{
				Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Target:  &ayd.URL{Scheme: "dummy"},
				Status:  ayd.StatusDegrade,
				Message: "",
				Latency: 1027890 * time.Microsecond,
			},
		},
		{
			String: `{"time":1641135845, "status":"healthy", "latency":12.345, "target":"dummy:"}`,
			Encode: `{"time":"2022-01-02T15:04:05Z", "status":"HEALTHY", "latency":12.345, "target":"dummy:"}`,
			Record: ayd.Record{
				Time:    time.Date(2022, 1, 2, 15, 4, 5, 0, time.UTC),
				Target:  &ayd.URL{Scheme: "dummy"},
				Status:  ayd.StatusHealthy,
				Message: "",
				Latency: 12345 * time.Microsecond,
			},
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123abc, "target":"ping:example.com", "message":"hello world"}`,
			Error:  "invalid record: invalid character 'a' after object key:value pair",
		},
		{
			String: `{"time":"2021/01/02 15:04:05", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world"}`,
			Error:  `invalid record: time: invalid format: "2021/01/02 15:04:05"`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"::invalid target::", "message":"hello world"}`,
			Error:  `invalid record: target: parse "::invalid target::": missing protocol scheme`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"://", "message":"hello world"}`,
			Error:  `invalid record: target: parse "://": missing protocol scheme`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"/", "message":"hello world"}`,
			Error:  `invalid record: target: parse "/": missing protocol scheme`,
		},
		{
			String: `{"status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world"}`,
			Error:  `invalid record: time: missing required field`,
		},
		{
			String: `{"time":{}, "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world"}`,
			Error:  `invalid record: time: should be a string or a number`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":null, "latency":123.456, "target":"ping:example.com", "message":"hello world"}`,
			Error:  `invalid record: status: should be a string`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":"hello", "target":"ping:example.com", "message":"hello world"}`,
			Error:  `invalid record: latency: should be a number`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "message":"hello world"}`,
			Error:  `invalid record: target: missing required field`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":1234, "message":"hello world"}`,
			Error:  `invalid record: target: should be a string`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com"}`,
			Record: ayd.Record{
				Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Status:  ayd.StatusHealthy,
				Latency: 123456 * time.Microsecond,
				Target:  &ayd.URL{Scheme: "ping", Opaque: "example.com"},
				Message: "",
			},
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":123}`,
			Error:  `invalid record: message: should be a string`,
		},
		{
			String: `{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world", "hello":"world"}`,
			Record: ayd.Record{
				Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, tokyo),
				Status:  ayd.StatusHealthy,
				Latency: 123456 * time.Microsecond,
				Target:  &ayd.URL{Scheme: "ping", Opaque: "example.com"},
				Message: "hello world",
				Extra: map[string]interface{}{
					"hello": "world",
				},
			},
		},
	}

	for _, tt := range tests {
		r, err := ayd.ParseRecord(tt.String)
		if tt.Error != "" {
			if err == nil {
				t.Errorf("expected %q error when parse %#v but got nil", tt.Error, tt.String)
			} else if diff := cmp.Diff(err.Error(), tt.Error); diff != "" {
				t.Errorf("unexpected error when parse %#v\n%s", tt.String, diff)
			}
			continue
		}
		if err != nil {
			t.Errorf("failed to parse %#v: %s", tt.String, err)
			continue
		}

		if !r.Time.Equal(tt.Record.Time) {
			t.Errorf("unexpected parsed timestamp\nexpected: %#v\n but got: %#v", tt.Record.Time, r.Time)
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

		if diff := cmp.Diff(tt.Record.Extra, r.Extra); diff != "" {
			t.Errorf("unexpected extra\n%s", diff)
		}

		expect := tt.Encode
		if expect == "" {
			expect = tt.String
		}
		if tt.Record.String() != expect {
			t.Errorf("expected: %#v\n but got: %#v", expect, tt.Record.String())
		}
	}
}

func TestRecord_redact(t *testing.T) {
	record := ayd.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Target:  &ayd.URL{Scheme: "http", Path: "/path/to/file", User: url.UserPassword("MyName", "HideMe")},
		Status:  ayd.StatusHealthy,
		Message: "hello world",
		Latency: 123456 * time.Microsecond,
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
			Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
			Status:  ayd.StatusHealthy,
			Latency: 123456 * time.Microsecond,
			Target:  &ayd.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
			Message: "this is test",
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
		source := `{"time":"2021-01-02T15:04:05Z", "status":"HEALTHY", "latency":123.456, "target":"dummy:healthy#hello-world", "message":"this is test", "extra":123, "hello":"world"}`
		expect := ayd.Record{
			Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
			Status:  ayd.StatusHealthy,
			Latency: 123456 * time.Microsecond,
			Target:  &ayd.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
			Message: "this is test",
			Extra: map[string]interface{}{
				"extra": 123.0,
				"hello": "world",
			},
		}

		var r ayd.Record
		if err := json.Unmarshal([]byte(source), &r); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if diff := cmp.Diff(r, expect); diff != "" {
			t.Fatalf("unexpected unmarshalled result\n%s", diff)
		}
	})
}

func TestRecord_ReadableMessage(t *testing.T) {
	tests := []struct {
		Message string
		Extra   map[string]interface{}
		Output  string
	}{
		{
			"",
			map[string]interface{}{
				"array":  []int{1, 2, 3},
				"hello":  "world",
				"multi":  "hello\nworld",
				"num":    42,
				"object": map[string]string{"key": "value"},
			},
			strings.Join([]string{
				"---",
				"array: [1,2,3]",
				"hello: world",
				"multi: |",
				"  hello",
				"  world",
				"num: 42",
				`object: {"key":"value"}`,
			}, "\n"),
		},
		{
			"hello\nworld",
			map[string]interface{}{
				"hello": "world",
			},
			strings.Join([]string{
				"hello",
				"world",
				"---",
				"hello: world",
			}, "\n"),
		},
		{
			"hello\nworld\n",
			map[string]interface{}{
				"hello": "world",
				"time":  "invalid key",
			},
			strings.Join([]string{
				"hello",
				"world",
				"---",
				"hello: world",
			}, "\n"),
		},
		{
			"hello world",
			nil,
			"hello world",
		},
		{
			"",
			nil,
			"",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual := ayd.Record{Message: tt.Message, Extra: tt.Extra}.ReadableMessage()
			if diff := cmp.Diff(actual, tt.Output); diff != "" {
				t.Errorf("unexpected output\n%s", diff)
			}
		})
	}
}

func TestRecord_MarshalJSON(t *testing.T) {
	tests := []struct {
		R ayd.Record
		S string
	}{
		{
			ayd.Record{
				Time:   time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
				Status: ayd.StatusHealthy,
				Target: &ayd.URL{Scheme: "dummy"},
				Extra: map[string]interface{}{
					"status": "invalid status",
					"foo":    "bar",
				},
			},
			`{"time":"2021-01-02T15:04:05Z", "status":"HEALTHY", "latency":0.000, "target":"dummy:", "foo":"bar"}`,
		},
		{
			ayd.Record{},
			`{"time":"0001-01-01T00:00:00Z", "status":"UNKNOWN", "latency":0.000, "target":""}`,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			b, err := tt.R.MarshalJSON()
			if err != nil {
				t.Fatalf("failed to marshal: %s", err)
			}
			if string(b) != tt.S {
				t.Errorf("unexpected result:\nexpected: %s\n but got: %s", tt.S, string(b))
			}
		})
	}
}

func BenchmarkRecord_MarshalJSON(b *testing.B) {
	record := ayd.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:  ayd.StatusHealthy,
		Latency: 123456 * time.Microsecond,
		Target:  &ayd.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
		Message: "this is test",
		Extra: map[string]interface{}{
			"extra": 123.0,
			"hello": "world",
		},
	}

	enc, err := record.MarshalJSON()
	if err != nil {
		b.Fatalf("failed to marshal: %s", err)
	}
	b.SetBytes(int64(len(enc)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = record.MarshalJSON()
	}
}

func BenchmarkRecord_UnmarshalJSON(b *testing.B) {
	bytes, err := ayd.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:  ayd.StatusHealthy,
		Latency: 123456 * time.Microsecond,
		Target:  &ayd.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
		Message: "this is test",
		Extra: map[string]interface{}{
			"extra": 123.0,
			"hello": "world",
		},
	}.MarshalJSON()

	if err != nil {
		b.Fatalf("failed to marshal: %s", err)
	}
	b.SetBytes(int64(len(bytes)))

	var record ayd.Record

	if err = record.UnmarshalJSON(bytes); err != nil {
		b.Fatalf("failed to unmarshal: %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = record.UnmarshalJSON(bytes)
	}
}
