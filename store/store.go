package store

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"sort"
	"sync"
)

const (
	PROBE_HISTORY_LEN    = 40
	INCIDENT_HISTORY_LEN = 10
	LOG_RESTORE_BYTES    = 1024 * 1024
)

type ProbeHistory struct {
	Target  *url.URL
	Results []*Record
}

type ProbeHistoryMap map[string]*ProbeHistory

func (hs ProbeHistoryMap) append(r *Record) {
	target := r.Target.String()

	if h, ok := hs[target]; ok {
		if len(h.Results) >= PROBE_HISTORY_LEN {
			h.Results = h.Results[1:]
		}

		h.Results = append(h.Results, r)
	} else {
		hs[target] = &ProbeHistory{
			Target:  r.Target,
			Results: []*Record{r},
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

func (s *Store) setIncidentIfNeed(r Record) {
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

	if r.Status != STATUS_OK {
		s.CurrentIncidents = append(s.CurrentIncidents, NewIncident(r))
	}
}

func (s *Store) Append(r Record) {
	s.Lock()
	defer s.Unlock()

	r = r.Sanitize()

	f, err := os.OpenFile(s.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %s", err)
		return
	}
	defer f.Close()

	str := r.String()
	fmt.Println(str)
	fmt.Fprintln(f, str)

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
	f.Seek(-LOG_RESTORE_BYTES, os.SEEK_END)

	s.ProbeHistory = make(ProbeHistoryMap)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		r, err := ParseRecord(scanner.Text())
		if err != nil {
			continue
		}

		s.ProbeHistory.append(&r)

		s.setIncidentIfNeed(r)
	}

	return nil
}
