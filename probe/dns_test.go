package probe_test

import (
	"context"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/testutil"
)

func TestDNSProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dns:localhost", api.StatusHealthy, "(127\\.0\\.0\\.1|::1)(\n(127\\.0\\.0\\.1|::1))*", ""},

		{"dns:localhost?type=AAAA", api.StatusHealthy, "::1(\n::1)*", ""},
		{"dns:localhost?type=A", api.StatusHealthy, "127\\.0\\.0\\.1(\n127\\.0\\.0\\.1)*", ""},

		{"dns:example.com?type=CNAME", api.StatusHealthy, `example.com.`, ""},

		{"dns:example.com?type=MX", api.StatusHealthy, `.`, ""},

		{"dns:example.com?type=NS", api.StatusHealthy, `[a-z]\.iana-servers\.net\.(` + "\n" + `[a-z]\.iana-servers\.net\.)*`, ""},

		{"dns:example.com?type=TXT", api.StatusHealthy, "(v=spf1 -all\n[0-9a-z]{32}|[0-9a-z]{32}\nv=spf1 -all)", ""},

		{"dns:of-course-definitely-no-such-host", api.StatusFailure, `lookup of-course-definitely-no-such-host(:| ).+`, ""},

		{"dns:example.com?type=UNKNOWN", api.StatusUnknown, ``, "unsupported DNS type"},
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
