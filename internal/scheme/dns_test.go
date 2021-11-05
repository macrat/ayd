package scheme_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestDNSProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dns:localhost", api.StatusHealthy, `ip=(127\.0\.0\.1|::1)(,(127\.0\.0\.1|::1))*`, ""},
		{"dns://8.8.8.8/localhost", api.StatusHealthy, `ip=(127\.0\.0\.1|::1)(,(127\.0\.0\.1|::1))*`, ""},
		{"dns://8.8.4.4:53/localhost", api.StatusHealthy, `ip=(127\.0\.0\.1|::1)(,(127\.0\.0\.1|::1))*`, ""},

		{"dns:localhost?type=AAAA", api.StatusHealthy, "ip=::1(,::1)*", ""},
		{"dns:localhost?type=A", api.StatusHealthy, `ip=127\.0\.0\.1(,127\.0\.0\.1)*`, ""},

		{"dns:example.com?type=CNAME", api.StatusHealthy, `hostname=example\.com\.`, ""},
		{"dns://1.1.1.1/example.com?type=CNAME", api.StatusHealthy, `hostname=example\.com\.`, ""},

		{"dns:google.com?type=MX", api.StatusHealthy, `mx=[a-z0-9.,]+`, ""},
		{"dns://8.8.8.8:53/google.com?type=MX", api.StatusHealthy, `mx=[a-z0-9.,]+`, ""},

		{"dns:example.com?type=NS", api.StatusHealthy, `ns=[a-z]\.iana-servers\.net\.(,[a-z]\.iana-servers\.net\.)*`, ""},
		{"dns://8.8.4.4/example.com?type=NS", api.StatusHealthy, `ns=[a-z]\.iana-servers\.net\.(,[a-z]\.iana-servers\.net\.)*`, ""},

		{"dns:example.com?type=TXT", api.StatusHealthy, "(v=spf1 -all\n[0-9a-z]{32}|[0-9a-z]{32}\nv=spf1 -all)", ""},
		{"dns://1.1.1.1/example.com?type=TXT", api.StatusHealthy, "(v=spf1 -all\n[0-9a-z]{32}|[0-9a-z]{32}\nv=spf1 -all)", ""},

		{"dns:example.com?type=UNKNOWN", api.StatusUnknown, ``, "unsupported DNS type"},
	}, 10)

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
