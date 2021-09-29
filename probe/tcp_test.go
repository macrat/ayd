package probe_test

import (
	"context"
	"strings"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/testutil"
)

func TestTCPProbe(t *testing.T) {
	t.Parallel()

	server := RunDummyHTTPServer()
	defer server.Close()

	AssertProbe(t, []ProbeTest{
		{strings.Replace(server.URL, "http://", "tcp://", 1), api.StatusHealthy, `source=(127\.0\.0\.1|\[::1\]):[0-9]+ target=(127\.0\.0\.1|\[::1\]):[0-9]+`, ""},
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
