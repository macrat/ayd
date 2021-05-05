package probe_test

import (
	"context"
	"testing"
	"time"

	"github.com/macrat/ayd/testutil"
)

func BenchmarkExecuteProbe(b *testing.B) {
	p := testutil.NewProbe(b, "exec:echo#hello-world")

	r := &testutil.DummyReporter{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(ctx, r)
	}
}
