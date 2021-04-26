package store

import (
	"bufio"
	"io"
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

func (hs ProbeHistoryMap) Append(r Record) {
	target := r.Target.String()

	if h, ok := hs[target]; ok {
		if len(h.Records) >= PROBE_HISTORY_LEN {
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

type IncidentHandler func(*Incident) []Record

type Store struct {
	sync.RWMutex

	Path string

	Console io.Writer

	probeHistory     ProbeHistoryMap
	currentIncidents map[string]*Incident
	incidentHistory  []*Incident

	OnIncident    []IncidentHandler
	IncidentCount int

	file      *os.File
	lastError error
}

func New(path string) (*Store, error) {
	store := &Store{
		Path:             path,
		Console:          os.Stdout,
		probeHistory:     make(ProbeHistoryMap),
		currentIncidents: make(map[string]*Incident),
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

func (s *Store) ProbeHistory() []*ProbeHistory {
	s.RLock()
	defer s.RUnlock()

	var targets []string
	for t := range s.probeHistory {
		targets = append(targets, t)
	}
	sort.Strings(targets)

	var result []*ProbeHistory
	for _, t := range targets {
		result = append(result, s.probeHistory[t])
	}

	return result
}

func (s *Store) CurrentIncidents() []*Incident {
	s.RLock()
	defer s.RUnlock()

	result := make([]*Incident, len(s.currentIncidents))

	i := 0
	for _, x := range s.currentIncidents {
		result[i] = x
		i++
	}

	sort.Sort(byIncidentCaused(result))

	return result
}

func (s *Store) IncidentHistory() []*Incident {
	return s.incidentHistory
}

func (s *Store) setIncidentIfNeed(r Record, needCallback bool) {
	target := r.Target.String()
	if cur, ok := s.currentIncidents[target]; ok {
		if cur.IsContinued(r) {
			return
		}

		cur.ResolvedAt = r.CheckedAt
		s.incidentHistory = append(s.incidentHistory, cur)
		delete(s.currentIncidents, target)

		if len(s.incidentHistory) > INCIDENT_HISTORY_LEN {
			s.incidentHistory = s.incidentHistory[1:]
		}
	}

	if r.Status != STATUS_HEALTHY {
		incident := NewIncident(r)
		s.currentIncidents[target] = incident

		if needCallback {
			s.IncidentCount++
			for _, cb := range s.OnIncident {
				s.appendWithoutLock(cb(incident))
			}
		}
	}
}

func (s *Store) appendWithoutLock(rs []Record) {
	for _, r := range rs {
		r = r.Sanitize()

		msg := []byte(r.String() + "\n")
		s.Console.Write(msg)
		_, s.lastError = s.file.Write(msg)

		if r.Target.Scheme != "alert" {
			s.probeHistory.Append(r)
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

	s.probeHistory = make(ProbeHistoryMap)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		r, err := ParseRecord(scanner.Text())
		if err != nil {
			continue
		}

		if r.Target.Scheme != "alert" {
			s.probeHistory.Append(r)
			s.setIncidentIfNeed(r, false)
		}
	}

	return nil
}

func (s *Store) AddTarget(target *url.URL) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.probeHistory[target.String()]; !ok {
		s.probeHistory[target.String()] = &ProbeHistory{
			Target: target,
		}
	}
}

func (s *Store) Err() error {
	return s.lastError
}
