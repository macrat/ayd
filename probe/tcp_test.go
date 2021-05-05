package probe_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/testutil"
)

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
