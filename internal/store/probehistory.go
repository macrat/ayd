package store

import (
	api "github.com/macrat/ayd/lib-ayd"
)

// probeHistory is a record history about single target.
//
// It also records reporter URLs that called as "source" to trace this target is active or not.
// See also isActive method.
type probeHistory struct {
	Target  *api.URL
	Records []api.Record
	sources []string
}

func (ph probeHistory) MakeReport(length int) api.ProbeHistory {
	l := len(ph.Records) - length
	if l < 0 {
		l = 0
	}

	r := api.ProbeHistory{
		Target:  ph.Target,
		Records: ph.Records[l:],
	}

	if len(ph.Records) > 0 {
		latest := ph.Records[len(ph.Records)-1]
		r.Status = latest.Status
		r.Updated = latest.CheckedAt
	}

	return r
}

// addSource appends reporter URL that reports to this probeHistory.
// The sources will used to detect if is this target active or not.
func (ph *probeHistory) addSource(source *api.URL) {
	s := source.String()
	for _, x := range ph.sources {
		if x == s {
			return
		}
	}

	ph.sources = append(ph.sources, s)
}

// removeSource removes reporter URL, that reports to this probeHistory, from sources.
func (ph *probeHistory) removeSource(source *api.URL) {
	s := source.String()
	for i, x := range ph.sources {
		if x == s {
			ph.sources = append(ph.sources[:i], ph.sources[i+1:]...)
			return
		}
	}
}

// setInactive removes all reporter URLs from this probeHistory.
func (ph *probeHistory) setInactive() {
	ph.sources = nil
}

// isActive returns if is this probeHistory active in current execution or not.
// Active means Ayd may append new record about the target, and inactive means Ayd won't append records unless the source or plugin changes.
func (ph probeHistory) isActive() bool {
	return len(ph.sources) != 0
}

// probeHistoryMap is a map of probeHistory.
// The key is a string of the target URL.
type probeHistoryMap map[string]*probeHistory

// Append adds ayd.Record to the probeHistory.
//
// `source` of argument means who is reporting this record.
// In the almost cases, it is the same as r.Target, but some cases like `source:` have another URL.
func (hs probeHistoryMap) Append(source *api.URL, r api.Record) {
	target := r.Target.String()

	if h, ok := hs[target]; ok {
		h.Records = append(h.Records, r)

		for i := len(h.Records) - 1; i > 0 && h.Records[i-1].CheckedAt.After(h.Records[i].CheckedAt); i-- {
			h.Records[i], h.Records[i-1] = h.Records[i-1], h.Records[i]
		}

		if len(h.Records) > PROBE_HISTORY_LEN {
			h.Records = h.Records[1:]
		}
	} else {
		hs[target] = &probeHistory{
			Target:  r.Target,
			Records: []api.Record{r},
		}
	}

	hs[target].addSource(source)
}

// isActive returns if is the specified target active in current execution or not.
func (hs probeHistoryMap) isActive(target *api.URL) bool {
	return hs[target.String()].isActive()
}
