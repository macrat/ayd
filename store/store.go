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
	Records []*Record
}

type ProbeHistoryMap map[string]*ProbeHistory

func (hs ProbeHistoryMap) append(r Record) {
	target := r.Target.String()

	if h, ok := hs[target]; ok {
		if len(h.Records) > PROBE_HISTORY_LEN {
			h.Records = h.Records[1:]
		}

		h.Records = append(h.Records, &r)
	} else {
		hs[target] = &ProbeHistory{
			Target:  r.Target,
			Records: []*Record{&r},
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

type IncidentHandler func(*Incident) []Record

type Store struct {
	sync.Mutex

	Path string

	ProbeHistory     ProbeHistoryMap
	CurrentIncidents []*Incident
	IncidentHistory  []*Incident

	OnIncident    []IncidentHandler
	IncidentCount int

	file      *os.File
	lastError error
}

func New(path string) (*Store, error) {
	store := &Store{
		Path:         path,
		ProbeHistory: make(ProbeHistoryMap),
	}

	var err error
	store.file, err = os.OpenFile(store.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0644)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.file.Close()
}

func (s *Store) setIncidentIfNeed(r Record, needCallback bool) {
	for i := 0; i < len(s.CurrentIncidents); i++ {
		x := s.CurrentIncidents[i]
		if x.SameTarget(r) {
			if !x.IsContinued(r) {
				x.ResolvedAt = r.CheckedAt
				s.IncidentHistory = append(s.IncidentHistory, x)
				s.CurrentIncidents = append(s.CurrentIncidents[:i], s.CurrentIncidents[i+1:]...)

				if len(s.IncidentHistory) > INCIDENT_HISTORY_LEN {
					s.IncidentHistory = s.IncidentHistory[1:]
				}

				break
			}

			return
		}
	}

	if r.Status != STATUS_HEALTHY {
		incident := NewIncident(r)
		s.CurrentIncidents = append(s.CurrentIncidents, incident)

		if needCallback {
			s.IncidentCount++
			for _, cb := range s.OnIncident {
				s.appendWithoutLock(cb(incident))
			}
		}
	}
}

func (s *Store) appendWithoutLock(rs []Record) {
	if s.file == nil {
		fmt.Fprintf(os.Stderr, "log file isn't opened. may be bug.")
		return
	}

	for _, r := range rs {
		r = r.Sanitize()

		str := r.String()
		fmt.Println(str)
		_, s.lastError = fmt.Fprintln(s.file, str)

		if r.Target.Scheme != "alert" {
			s.ProbeHistory.append(r)
			s.setIncidentIfNeed(r, true)
		}
	}
}

func (s *Store) Append(rs ...Record) {
	s.Lock()
	defer s.Unlock()

	s.appendWithoutLock(rs)
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

		if r.Target.Scheme != "alert" {
			s.ProbeHistory.append(r)
			s.setIncidentIfNeed(r, false)
		}
	}

	return nil
}

func (s *Store) AddTarget(target *url.URL) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.ProbeHistory[target.String()]; !ok {
		s.ProbeHistory[target.String()] = &ProbeHistory{
			Target: target,
		}
	}
}

func (s *Store) Err() error {
	return s.lastError
}
