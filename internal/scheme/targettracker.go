package scheme

import (
	"sync"

	api "github.com/macrat/ayd/lib-ayd"
)

// FixedSourceReporter is a Reporter that overrides source argument.
//
// This struct is used by TargetTracker.
type FixedSourceReporter struct {
	Source    *api.URL
	Upstreams []Reporter
}

// Report implements Reporter.
// This method just reports to upstream reporters.
func (r FixedSourceReporter) Report(_ *api.URL, rec api.Record) {
	for _, u := range r.Upstreams {
		u.Report(r.Source, rec)
	}
}

// DeactivateTarget implements Reporter.
func (r FixedSourceReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	for _, u := range r.Upstreams {
		u.DeactivateTarget(r.Source, targets...)
	}
}

// TargetTracker tracks the targets is active or not.
type TargetTracker struct {
	sync.Mutex

	actives   []*api.URL
	inactives []*api.URL
}

// PrepareReporter prepares to tracking a new probe with a new reporter.
func (t *TargetTracker) PrepareReporter(source *api.URL, r Reporter) Reporter {
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
func (t *TargetTracker) Inactives() []*api.URL {
	t.Lock()
	defer t.Unlock()

	return append([]*api.URL{}, t.inactives...)
}

func (t *TargetTracker) addActive(target *api.URL) {
	tgt := target.String()
	for _, a := range t.actives {
		if a.String() == tgt {
			return
		}
	}
	t.actives = append(t.actives, target)
}

func (t *TargetTracker) removeInactive(target *api.URL) {
	tgt := target.String()
	for i, a := range t.inactives {
		if a.String() == tgt {
			t.inactives = append(t.inactives[:i], t.inactives[i+1:]...)
			return
		}
	}
}

// Activate removes target URL from inactive list and appends to active list.
func (t *TargetTracker) Activate(target *api.URL) {
	t.Lock()
	defer t.Unlock()

	t.addActive(target)
	t.removeInactive(target)
}

// Report implements Reporter.
// This method marks as the target is active.
func (t *TargetTracker) Report(_ *api.URL, rec api.Record) {
	t.Activate(rec.Target)
}

// DeactivateTarget implements Reporter. it does nothing.
func (t *TargetTracker) DeactivateTarget(_ *api.URL, _ ...*api.URL) {
}
