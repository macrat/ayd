package testutil

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
)

type DummyReporter struct {
	sync.Mutex

	Records []api.Record
	Sources []*api.URL
	Actives []*api.URL
}

func (r *DummyReporter) Report(source *api.URL, rec api.Record) {
	r.Lock()
	defer r.Unlock()

	r.Records = append(r.Records, rec)
	r.Sources = append(r.Sources, source)

	for _, a := range r.Actives {
		if a.String() == rec.Target.String() {
			return
		}
	}
	r.Actives = append(r.Actives, rec.Target)
}

func (r *DummyReporter) removeTarget(t *api.URL) {
	for i, a := range r.Actives {
		if a.String() == t.String() {
			r.Actives = append(r.Actives[:i], r.Actives[i+1:]...)
			return
		}
	}
}

func (r *DummyReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	r.Lock()
	defer r.Unlock()

	for _, t := range targets {
		r.removeTarget(t)
	}
}

func (r *DummyReporter) AssertActives(t *testing.T, expects ...string) {
	t.Helper()

	sort.Strings(expects)

	as := []string{}
	for _, a := range r.Actives {
		as = append(as, a.String())
	}
	sort.Strings(as)

	if diff := cmp.Diff(expects, as); diff != "" {
		t.Errorf("unexpected active targets:\n%s", diff)
	}
}

func RunProbe(ctx context.Context, p scheme.Prober) []api.Record {
	reporter := &DummyReporter{}
	p.Probe(ctx, reporter)
	return reporter.Records
}

func RunAlert(ctx context.Context, a scheme.Alerter, rec api.Record) []api.Record {
	reporter := &DummyReporter{}
	a.Alert(ctx, reporter, rec)
	return reporter.Records
}
