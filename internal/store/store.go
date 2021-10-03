package store

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

const (
	PROBE_HISTORY_LEN    = 40
	INCIDENT_HISTORY_LEN = 10
)

var (
	LogRestoreBytes int64 = 100 * 1024 * 1024
)

type ProbeHistory struct {
	Target  *url.URL
	Records []api.Record
	shown   bool
}

func (ph ProbeHistory) MakeReport() api.ProbeHistory {
	r := api.ProbeHistory{
		Target:  ph.Target,
		Records: ph.Records,
	}

	if len(ph.Records) > 0 {
		latest := ph.Records[len(ph.Records)-1]
		r.Status = latest.Status
		r.Updated = latest.CheckedAt
	}

	return r
}

type byLatestStatus []*ProbeHistory

func (xs byLatestStatus) Len() int {
	return len(xs)
}

func statusTier(p *ProbeHistory) int {
	if len(p.Records) == 0 {
		return 1
	}
	switch p.Records[len(p.Records)-1].Status {
	case api.StatusFailure, api.StatusUnknown:
		return 0
	default:
		return 1
	}
}

func (xs byLatestStatus) Less(i, j int) bool {
	iTier := statusTier(xs[i])
	jTier := statusTier(xs[j])
	if iTier < jTier {
		return true
	} else if iTier > jTier {
		return false
	}

	return strings.Compare(xs[i].Target.Redacted(), xs[j].Target.Redacted()) < 0
}

func (xs byLatestStatus) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

type ProbeHistoryMap map[string]*ProbeHistory

func (hs ProbeHistoryMap) Append(r api.Record) {
	target := r.Target.Redacted()

	if h, ok := hs[target]; ok {
		h.Records = append(h.Records, r)

		for i := len(h.Records) - 1; i > 0 && h.Records[i-1].CheckedAt.After(h.Records[i].CheckedAt); i-- {
			h.Records[i], h.Records[i-1] = h.Records[i-1], h.Records[i]
		}

		if len(h.Records) > PROBE_HISTORY_LEN {
			h.Records = h.Records[1:]
		}
	} else {
		hs[target] = &ProbeHistory{
			Target:  r.Target,
			Records: []api.Record{r},
		}
	}

	hs[target].shown = true
}

func (hs ProbeHistoryMap) isShown(target *url.URL) bool {
	return hs[target.Redacted()].shown
}

type IncidentHandler func(*api.Incident)

type Store struct {
	Path string

	Console io.Writer

	historyLock      sync.RWMutex
	probeHistory     ProbeHistoryMap
	currentIncidents map[string]*api.Incident
	incidentHistory  []*api.Incident

	OnIncident    []IncidentHandler
	IncidentCount int

	writeCh       chan<- api.Record
	writerStopped chan struct{}
	errorLock     sync.RWMutex
	lastError     error
}

func New(path string) (*Store, error) {
	ch := make(chan api.Record, 32)

	store := &Store{
		Path:             path,
		Console:          os.Stdout,
		probeHistory:     make(ProbeHistoryMap),
		currentIncidents: make(map[string]*api.Incident),
		writeCh:          ch,
		writerStopped:    make(chan struct{}),
	}

	if store.Path != "" {
		if f, err := os.OpenFile(store.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0644); err != nil {
			close(ch)
			return nil, err
		} else {
			f.Close()
		}
	}

	go store.writer(ch, store.writerStopped)

	return store, nil
}

func (s *Store) ReportInternalError(scope, message string) {
	s.Report(api.Record{
		CheckedAt: time.Now(),
		Status:    api.StatusFailure,
		Target:    &url.URL{Scheme: "ayd", Opaque: scope},
		Message:   message,
	})
}

func (s *Store) handleError(err error) {
	if err != nil {
		s.Console.Write([]byte(api.Record{
			CheckedAt: time.Now(),
			Status:    api.StatusFailure,
			Target:    &url.URL{Scheme: "ayd", Opaque: "log"},
			Message:   err.Error(),
		}.String() + "\n"))
	}
}

func (s *Store) writer(ch <-chan api.Record, stopped chan struct{}) {
	for r := range ch {
		msg := []byte(r.String() + "\n")

		s.Console.Write(msg)

		if s.Path == "" {
			continue
		}

		s.errorLock.Lock()

		var f *os.File
		f, s.lastError = os.OpenFile(s.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if s.lastError != nil {
			s.handleError(s.lastError)
			s.errorLock.Unlock()
			continue
		}
		_, s.lastError = f.Write(msg)
		s.handleError(s.lastError)
		s.handleError(f.Close())

		s.errorLock.Unlock()
	}

	close(stopped)
}

func (s *Store) Close() error {
	close(s.writeCh)
	<-s.writerStopped
	return nil
}

func (s *Store) ProbeHistory() []*ProbeHistory {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	var result []*ProbeHistory
	for _, x := range s.probeHistory {
		if x.shown {
			result = append(result, x)
		}
	}

	sort.Sort(byLatestStatus(result))

	return result
}

func (s *Store) currentIncidentsWithoutLock() []*api.Incident {
	result := make([]*api.Incident, 0, len(s.currentIncidents))

	for _, x := range s.currentIncidents {
		if s.probeHistory.isShown(x.Target) {
			result = append(result, x)
		}
	}

	sort.Sort(byIncidentCaused(result))

	return result
}

func (s *Store) CurrentIncidents() []*api.Incident {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	return s.currentIncidentsWithoutLock()
}

func (s *Store) incidentHistoryWithoutLock() []*api.Incident {
	result := make([]*api.Incident, 0, len(s.incidentHistory))

	for _, x := range s.incidentHistory {
		if s.probeHistory.isShown(x.Target) {
			result = append(result, x)
		}
	}

	return result
}

func (s *Store) IncidentHistory() []*api.Incident {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	return s.incidentHistoryWithoutLock()
}

func (s *Store) setIncidentIfNeed(r api.Record, needCallback bool) {
	if r.Status == api.StatusAborted {
		return
	}

	target := r.Target.Redacted()
	if cur, ok := s.currentIncidents[target]; ok {
		if IncidentIsContinued(cur, r) {
			return
		}

		cur.ResolvedAt = r.CheckedAt
		s.incidentHistory = append(s.incidentHistory, cur)
		delete(s.currentIncidents, target)

		if len(s.incidentHistory) > INCIDENT_HISTORY_LEN {
			s.incidentHistory = s.incidentHistory[1:]
		}

		if needCallback {
			for _, cb := range s.OnIncident {
				cb(cur)
			}
		}
	}

	if r.Status != api.StatusHealthy {
		incident := NewIncident(r)
		s.currentIncidents[target] = incident

		if needCallback {
			s.IncidentCount++
			for _, cb := range s.OnIncident {
				cb(incident)
			}
		}
	}
}

func (s *Store) Report(r api.Record) {
	if _, ok := r.Target.User.Password(); ok {
		r.Target.User = url.UserPassword(r.Target.User.Username(), "xxxxx")
	}
	r.Message = strings.Trim(r.Message, "\r\n")

	s.writeCh <- r

	if r.Target.Scheme != "alert" && r.Target.Scheme != "ayd" {
		s.historyLock.Lock()
		defer s.historyLock.Unlock()

		s.probeHistory.Append(r)
		s.setIncidentIfNeed(r, true)
	}
}

func (s *Store) Restore() error {
	if s.Path == "" {
		return nil
	}

	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	f, err := os.OpenFile(s.Path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if ret, _ := f.Seek(-LogRestoreBytes, os.SEEK_END); ret != 0 {
		fmt.Fprint(os.Stderr, "WARNING: read only last 100MB from log file because it is too large\n\n")
	}

	s.probeHistory = make(ProbeHistoryMap)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		r, err := api.ParseRecord(scanner.Text())
		if err != nil {
			continue
		}

		if _, ok := r.Target.User.Password(); ok {
			r.Target.User = url.UserPassword(r.Target.User.Username(), "xxxxx")
		}

		if r.Target.Scheme != "alert" && r.Target.Scheme != "ayd" {
			s.probeHistory.Append(r)
			s.setIncidentIfNeed(r, false)
		}
	}

	for k := range s.probeHistory {
		s.probeHistory[k].shown = false
	}

	return nil
}

func (s *Store) AddTarget(target *url.URL) {
	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	t := target.Redacted()

	if _, ok := s.probeHistory[t]; !ok {
		s.probeHistory[t] = &ProbeHistory{
			Target: target,
		}
	}
	s.probeHistory[t].shown = true
}

func (s *Store) Err() error {
	s.errorLock.RLock()
	defer s.errorLock.RUnlock()
	return s.lastError
}

func (s *Store) MakeReport() api.Report {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	ci := s.currentIncidentsWithoutLock()
	ih := s.incidentHistoryWithoutLock()

	report := api.Report{
		ProbeHistory:     make(map[string]api.ProbeHistory),
		CurrentIncidents: make([]api.Incident, len(ci)),
		IncidentHistory:  make([]api.Incident, len(ih)),
		ReportedAt:       time.Now(),
	}

	for i, x := range ci {
		report.CurrentIncidents[i] = *x
	}

	for i, x := range ih {
		report.IncidentHistory[i] = *x
	}

	for k, v := range s.probeHistory {
		if v.shown {
			report.ProbeHistory[k] = v.MakeReport()
		}
	}

	return report
}
