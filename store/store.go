package store

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/macrat/ayd/probe"
)

const (
	PROBE_HISTORY_LEN    = 40
	INCIDENT_HISTORY_LEN = 10
)

type ProbeHistory struct {
	Target  *url.URL
	Results []*probe.Result
}

type ProbeHistoryMap map[string]*ProbeHistory

func (hs ProbeHistoryMap) append(r *probe.Result) {
	target := r.Target.String()

	if h, ok := hs[target]; ok {
		if len(h.Results) >= PROBE_HISTORY_LEN {
			h.Results = h.Results[1:]
		}

		h.Results = append(h.Results, r)
	} else {
		hs[target] = &ProbeHistory{
			Target:  r.Target,
			Results: []*probe.Result{r},
		}
	}
}

func (hs ProbeHistoryMap) AsSortedArray() []*ProbeHistory {
	var targets []string
	for t := range hs {
		targets = append(targets, t)
	}
	sort.Strings(targets)

	var result []*ProbeHistory
	for _, t := range targets {
		result = append(result, hs[t])
	}

	return result
}

type Store struct {
	sync.Mutex

	Path string

	ProbeHistory     ProbeHistoryMap
	CurrentIncidents []*Incident
	IncidentHistory  []*Incident
}

func New(path string) *Store {
	return &Store{
		Path:         path,
		ProbeHistory: make(ProbeHistoryMap),
	}
}

func (s *Store) setIncidentIfNeed(r probe.Result) {
	for i := 0; i < len(s.CurrentIncidents); i++ {
		x := s.CurrentIncidents[i]
		if x.SameTarget(r) {
			if !x.IsContinued(r) {
				x.ResolvedAt = r.CheckedAt
				s.IncidentHistory = append(s.IncidentHistory, x)
				s.CurrentIncidents = append(s.CurrentIncidents[:i], s.CurrentIncidents[i+1:]...)

				if len(s.IncidentHistory) >= INCIDENT_HISTORY_LEN {
					s.IncidentHistory = s.IncidentHistory[1:]
				}
			}

			return
		}
	}

	if r.Status != probe.STATUS_OK {
		s.CurrentIncidents = append(s.CurrentIncidents, NewIncident(r))
	}
}

func (s *Store) Append(r probe.Result) {
	s.Lock()
	defer s.Unlock()

	f, err := os.OpenFile(s.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %s", err)
		return
	}
	defer f.Close()

	fmt.Println(result2str(r, true))
	fmt.Fprintln(f, result2str(r, false))

	s.ProbeHistory.append(&r)

	s.setIncidentIfNeed(r)
}

func (s *Store) Restore() error {
	s.Lock()
	defer s.Unlock()

	f, err := os.OpenFile(s.Path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	s.ProbeHistory = make(ProbeHistoryMap)

	threshold := time.Now().Add(-24 * time.Hour)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		r, err := str2result(scanner.Text())
		if err != nil {
			continue
		}

		if threshold.After(r.CheckedAt) {
			continue
		}

		s.ProbeHistory.append(&r)

		s.setIncidentIfNeed(r)
	}

	return nil
}
