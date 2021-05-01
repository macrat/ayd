package testutil

import (
	"context"
	"sync"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

type DummyReporter struct {
	sync.Mutex

	Records []store.Record
}

func (r *DummyReporter) Report(rec store.Record) {
	r.Lock()
	defer r.Unlock()

	r.Records = append(r.Records, rec)
}

func RunCheck(ctx context.Context, p probe.Probe) []store.Record {
	reporter := &DummyReporter{}
	p.Check(ctx, reporter)
	return reporter.Records
}
