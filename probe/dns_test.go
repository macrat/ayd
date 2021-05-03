package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestDNSProbe(t *testing.T) {
	t.Parallel()

	AssertProbe(t, []ProbeTest{
		{"dns:localhost", store.STATUS_HEALTHY, "(127\\.0\\.0\\.1|::1)(\n(127\\.0\\.0\\.1|::1))*"},

		{"dns:localhost?type=AAAA", store.STATUS_HEALTHY, "::1(\n::1)*"},
		{"dns:localhost?type=A", store.STATUS_HEALTHY, "127\\.0\\.0\\.1(\n127\\.0\\.0\\.1)*"},

		{"dns:example.com?type=CNAME", store.STATUS_HEALTHY, `example.com.`},

		{"dns:example.com?type=MX", store.STATUS_HEALTHY, `.`},

		{"dns:example.com?type=NS", store.STATUS_HEALTHY, `[a-z]\.iana-servers\.net\.(` + "\n" + `[a-z]\.iana-servers\.net\.)*`},

		{"dns:example.com?type=TXT", store.STATUS_HEALTHY, "(v=spf1 -all\n[0-9a-z]{32}|[0-9a-z]{32}\nv=spf1 -all)"},

		{"dns:of-course-definitely-no-such-host", store.STATUS_FAILURE, `lookup of-course-definitely-no-such-host(:| ).+`},
	})

	AssertTimeout(t, "dns:localhost")
}
