package probe_test

import (
	"testing"

	"github.com/macrat/ayd/store"
)

func TestDNSProbe(t *testing.T) {
	AssertProbe(t, []ProbeTest{
		{"dns:localhost", store.STATUS_HEALTHY, `(127\.0\.0\.1|::1|127\.0\.0\.1, ::1|::1, 127\.0\.0\.1)`},
		{"dns:of-course-definitely-no-such-host", store.STATUS_FAILURE, `lookup of-course-definitely-no-such-host on .+: no such host`},
	})
}
