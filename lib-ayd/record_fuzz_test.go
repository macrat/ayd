package ayd_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/lib-ayd"
)

func FuzzParseRecord(f *testing.F) {
	f.Add(`{"time":"2021-01-02T15:04:05+09:00", "status":"HEALTHY", "latency":123.456, "target":"ping:example.com", "message":"hello world"}`)
	f.Add(`{"time":"2021-01-02_15:04:05+09:00", "status":"FAILURE", "latency":123.456, "target":"exec:/path/to/file.sh", "message":"hello world"}`)
	f.Add(`{"time":"2021-01-02 15:04:05+09", "status":"ABORTED", "latency":1234.567, "target":"dummy:#hello", "message":"hello world", "abc":["def","ghi"]}`)
	f.Add(`{"time":"2021-01-02T15:04:05+0900", "status":"DEGRADE", "latency":1.234, "target":"dummy:"}`)
	f.Add(`{"time":"20210102T150405+09:00", "status":"DEGRADE", "latency":1.234, "target":"dummy:", "extra":123.456, "hello":"world"}`)
	f.Add(`{"time":"2001-02-03T04:05:06-10:00", "status":"HEALTHY", "latency":1234.456, "target":"https://example.com/path/to/healthz", "message":"hello\tworld"}`)
	f.Add(`{"time":"1234-10-30T22:33:44Z", "status":"FAILURE", "latency":0.123, "target":"source+http://example.com/hello/world", "message":"this is test\nhello", "extra":123}`)
	f.Add(`{"time":"2000-10-23T14:56:37Z", "status":"ABORTED", "latency":987654.321, "target":"alert:foobar:alert-url", "message":"cancelled"}`)
	f.Add(`{"time":"2345-12-31T23:59:59.999-11:59","status":"UNKNOWN","latency":-1,"target":"http://@oh-no"}`)
	f.Add(`{"target":"http://:@oh-nyo","status":"failure","time":"2345-12-31T23:59:59.999-11:59","latency":-1}`)
	f.Add(`{ "target":"dummy:abc", "status":"Failure", "time":1234, "message" : "\xf2" }`)

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
		r.Latency = r.Latency.Round(time.Microsecond)
		r2.Latency = r.Latency.Round(time.Microsecond)
		r.Time = r.Time.Round(time.Second)
		r2.Time = r.Time.Round(time.Second)

		if diff := cmp.Diff(r, r2, cmp.Comparer(compareUserinfo)); diff != "" {
			t.Logf("input: %s", data)
			t.Logf("1st parse: %#v", r)
			t.Logf("1st parse URL: %s (%#v)", *r.Target, *r.Target)
			t.Logf("1st marshal: %s", s)
			t.Logf("2nd parse: %#v", r2)
			t.Logf("2nd parse URL: %s (%#v)", *r2.Target, *r2.Target)
			t.Logf("2nd marshal: %s", r2.String())
			t.Errorf("first generated and regenerated was different\n%s", diff)
		}
	})
}

func compareUserinfo(x, y *url.Userinfo) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	_, xok := x.Password()
	_, yok := y.Password()
	return x.Username() == y.Username() && xok == yok
}
