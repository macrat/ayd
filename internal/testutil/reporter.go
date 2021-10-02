package testutil

import (
	"context"
	"sync"

	"github.com/macrat/ayd/internal/probe"
	api "github.com/macrat/ayd/lib-ayd"
)

type DummyReporter struct {
	sync.Mutex

	Records []api.Record
}

func (r *DummyReporter) Report(rec api.Record) {
	r.Lock()
	defer r.Unlock()

	r.Records = append(r.Records, rec)
}

func RunCheck(ctx context.Context, p probe.Probe) []api.Record {
	reporter := &DummyReporter{}
	p.Check(ctx, reporter)
	return reporter.Records
}
