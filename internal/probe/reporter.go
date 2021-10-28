package probe

import (
	"net/url"
	"sync"

	api "github.com/macrat/ayd/lib-ayd"
)

type Reporter interface {
	// Report reports a Record.
	//
	// `source` in argument is the probe's URL.
	Report(source *url.URL, r api.Record)

	// DeactivateTarget marks the target is no longer reported via specified source.
	DeactivateTarget(source *url.URL, targets ...*url.URL)
}

// FixedSourceReporter is a Reporter that overrides source argument.
//
// This struct is used by TargetTracker.
type FixedSourceReporter struct {
	Source    *url.URL
	Upstreams []Reporter
}

// Report implements Reporter.
// This method just reports to upstream reporters.
func (r FixedSourceReporter) Report(_ *url.URL, rec api.Record) {
	for _, u := range r.Upstreams {
		u.Report(r.Source, rec)
	}
}

// DeactivateTarget implements Reporter.
func (r FixedSourceReporter) DeactivateTarget(source *url.URL, targets ...*url.URL) {
	for _, u := range r.Upstreams {
		u.DeactivateTarget(r.Source, targets...)
	}
}

// TargetTracker tracks the targets is active or not.
type TargetTracker struct {
	sync.Mutex

	actives   []*url.URL
	inactives []*url.URL
}

// PrepareReporter prepares to tracking a new probe with a new reporter.
func (t *TargetTracker) PrepareReporter(source *url.URL, r Reporter) Reporter {
	t.Lock()
	defer t.Unlock()

	t.inactives = t.actives
	t.actives = nil

	return FixedSourceReporter{
		Source:    source,
		Upstreams: []Reporter{r, t},
	}
}

// Inactives returns the list of inactive targets that not reported since last PrepareReporter called.
func (t *TargetTracker) Inactives() []*url.URL {
	t.Lock()
	defer t.Unlock()

	return append([]*url.URL{}, t.inactives...)
}

func (t *TargetTracker) addActive(target *url.URL) {
	tgt := target.Redacted()
	for _, a := range t.actives {
		if a.Redacted() == tgt {
			return
		}
	}
	t.actives = append(t.actives, target)
}

func (t *TargetTracker) removeInactive(target *url.URL) {
	tgt := target.Redacted()
	for i, a := range t.inactives {
		if a.Redacted() == tgt {
			t.inactives = append(t.inactives[:i], t.inactives[i+1:]...)
			return
		}
	}
}

// Activate removes target URL from inactive list and appends to active list.
func (t *TargetTracker) Activate(target *url.URL) {
	t.Lock()
	defer t.Unlock()

	t.addActive(target)
	t.removeInactive(target)
}

// Report implements Reporter.
// This method marks as the target is active.
func (t *TargetTracker) Report(_ *url.URL, rec api.Record) {
	t.Activate(rec.Target)
}

// DeactivateTarget implements Reporter. it does nothing.
func (t *TargetTracker) DeactivateTarget(_ *url.URL, _ ...*url.URL) {
}
