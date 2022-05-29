//go:build go1.18
// +build go1.18

package ayd_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/lib-ayd"
)

func FuzzParseRecord(f *testing.F) {
	f.Add(`{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world"}`)
	f.Add(`{"time":"2021-01-02T15:04:05+09:00", "status":"FAILURE", "latency":123.456, "target":"exec:/path/to/file.sh", "message":"hello world"}`)
	f.Add(`{"time":"2021-01-02T15:04:05+09:00", "status":"ABORTED", "latency":1234.567, "target":"dummy:#hello", "message":"hello world"}`)
	f.Add(`{"time":"2021-01-02T15:04:05+09:00", "status":"DEGRADE", "latency":1.234, "target":"dummy:"}`)
	f.Add(`{"time":"2021-01-02T15:04:05+09:00", "status":"DEGRADE", "latency":1.234, "target":"dummy:", "extra":123.456, "hello":"world"}`)
	f.Add(`{"time":"2001-02-03T04:05:06-10:00", "status":"HEALTHY", "latency":1234.456, "target":"https://example.com/path/to/healthz", "message":"hello\tworld"}`)
	f.Add(`{"time":"1234-10-30T22:33:44Z", "status":"FAILURE", "latency":0.123, "target":"source+http://example.com/hello/world", "message":"this is test\nhello"}`)
	f.Add(`{"time":"2000-10-23T14:56:37Z", "status":"ABORTED", "latency":987654.321, "target":"alert:foobar:alert-url", "message":"cancelled"}`)

	f.Fuzz(func(t *testing.T, data string) {
		r, err := ayd.ParseRecord(data)
		if err != nil {
			t.Skip()
		}

		s := r.String()

		r2, err := ayd.ParseRecord(s)
		if err != nil {
			t.Fatalf("failed to parse again: %s", err)
		}

		r.Target.RawPath = ""
		r2.Target.RawPath = ""
		r.Target.RawFragment = ""
		r2.Target.RawFragment = ""

		if diff := cmp.Diff(r, r2); diff != "" {
			t.Errorf("first generated and regenerated was different\n%s", diff)
		}
	})
}
