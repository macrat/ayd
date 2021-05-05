package probe_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestDNSProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dns:localhost", store.STATUS_HEALTHY, "(127\\.0\\.0\\.1|::1)(\n(127\\.0\\.0\\.1|::1))*", ""},

		{"dns:localhost?type=AAAA", store.STATUS_HEALTHY, "::1(\n::1)*", ""},
		{"dns:localhost?type=A", store.STATUS_HEALTHY, "127\\.0\\.0\\.1(\n127\\.0\\.0\\.1)*", ""},

		{"dns:example.com?type=CNAME", store.STATUS_HEALTHY, `example.com.`, ""},

		{"dns:example.com?type=MX", store.STATUS_HEALTHY, `.`, ""},

		{"dns:example.com?type=NS", store.STATUS_HEALTHY, `[a-z]\.iana-servers\.net\.(` + "\n" + `[a-z]\.iana-servers\.net\.)*`, ""},

		{"dns:example.com?type=TXT", store.STATUS_HEALTHY, "(v=spf1 -all\n[0-9a-z]{32}|[0-9a-z]{32}\nv=spf1 -all)", ""},

		{"dns:of-course-definitely-no-such-host", store.STATUS_FAILURE, `lookup of-course-definitely-no-such-host(:| ).+`, ""},

		{"dns:example.com?type=UNKNOWN", store.STATUS_UNKNOWN, ``, "unsupported DNS type"},
	})

	AssertTimeout(t, "dns:localhost")
}

func BenchmarkDNSProbe(b *testing.B) {
	p := testutil.NewProbe(b, "dns:localhost")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	r := &testutil.DummyReporter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
