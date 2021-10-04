package probe_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestTCPProbe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{strings.Replace(server.URL, "http://", "tcp://", 1), api.StatusHealthy, `source=(127\.0\.0\.1|\[::1\]):[0-9]+ target=(127\.0\.0\.1|\[::1\]):[0-9]+`, ""},
		{"tcp://of-course-no-such-host.local:54321", api.StatusUnknown, "lookup of-course-no-such-host.local: host not found", ""},

		{"tcp://localhost", api.StatusUnknown, ``, "TCP target's port number is required"},
	})
}

func BenchmarkTCPProbe(b *testing.B) {
	server := RunDummyHTTPServer()
	defer server.Close()

	p := testutil.NewProbe(b, strings.Replace(server.URL, "http:", "tcp:", 1))

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
