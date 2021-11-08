//go:build gofuzzbeta
// +build gofuzzbeta

package ayd_test

import (
	"testing"

	"github.com/macrat/ayd/lib-ayd"
)

func FuzzParseRecord(f *testing.F) {
	f.Add("2021-01-02T15:04:05+09:00\tHEALTHY\t123.456\tping:example.com\thello world")
	f.Add("2021-01-02T15:04:05+09:00\tFAILURE\t123.456\texec:/path/to/file.sh\thello world")
	f.Add("2021-01-02T15:04:05+09:00\tABORTED\t1234.567\tdummy:#hello\thello world")
	f.Add("2021-01-02T15:04:05+09:00\tDEBASED\t1.234\tdummy:\t")
	f.Add("2001-02-03T04:05:06-10:00\tHEALTHY\t1234.456\thttps://example.com/path/to/healthz\thello\\tworld")
	f.Add("1234-10-30T22:33:44Z\tFAILURE\t0.123\tsource+http://example.com/hello/world\tthis is test\\nhello")
	f.Add("2000-10-23T14:56:37Z\tABORTED\t987654.321\talert:foobar:alert-url\tcancelled")

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

		s2 := r2.String()

		if s != s2 {
			t.Errorf("first generated and regenerated was different\n1st: %q\n2nd: %q", s, s2)
		}
	})
}
