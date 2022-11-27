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
	PROBE_HISTORY_LEN    = 60
	INCIDENT_HISTORY_LEN = 20
)

var (
	LogRestoreBytes int64 = 1024 * 1024 * 1024
)

type RecordHandler func(api.Record)

// Store is the log handler of Ayd, and it also the database of Ayd.
type Store struct {
	path string

	Console io.Writer

	historyLock      sync.RWMutex
	probeHistory     probeHistoryMap
	currentIncidents map[string]*api.Incident
	incidentHistory  []*api.Incident
	index            *indexer

	OnStatusChanged []RecordHandler
	incidentCount   int

	writeCh       chan<- api.Record
	writerStopped chan struct{}
	errorsLock    sync.RWMutex
	errors        []string
	healthy       bool
}

func New(path string, console io.Writer) (*Store, error) {
	ch := make(chan api.Record, 32)

	store := &Store{
		path:             path,
		Console:          console,
		probeHistory:     make(probeHistoryMap),
		currentIncidents: make(map[string]*api.Incident),
		index:            newIndexer(),
		writeCh:          ch,
		writerStopped:    make(chan struct{}),
		healthy:          true,
	}

	if store.path != "" {
		if f, err := os.OpenFile(store.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0644); err != nil {
			close(ch)
			return nil, err
		} else {
			f.Close()
		}
	}

	go store.writer(ch, store.writerStopped)

	return store, nil
}

// Path returns path to log file.
func (s *Store) Path() string {
	return s.path
}

// SetPath sets path to log file.
func (s *Store) SetPath(p string) {
	s.path = p
}

// IncidentCount returns the count of incident causes.
func (s *Store) IncidentCount() int {
	return s.incidentCount
}

func (s *Store) ReportInternalError(scope, message string) {
	u := &api.URL{Scheme: "ayd", Opaque: scope}

	s.Report(u, api.Record{
		Time:    time.Now(),
		Status:  api.StatusFailure,
		Target:  u,
		Message: message,
	})
}

// handleError reports an error of write a log.
// This error will reported to console in this method, and /healthz page via Store.Errors method.
func (s *Store) handleError(err error, exportableErrorMessage string) {
	if err != nil {
		s.addError(exportableErrorMessage)
		strings.NewReader(api.Record{
			Time:    time.Now(),
			Status:  api.StatusFailure,
			Target:  &api.URL{Scheme: "ayd", Opaque: "log"},
			Message: err.Error(),
		}.String() + "\n").WriteTo(s.Console)
	}
}

func (s *Store) writer(ch <-chan api.Record, stopped chan struct{}) {
	var reader strings.Reader

	for r := range ch {
		msg := r.String() + "\n"

		reader.Reset(msg)
		reader.WriteTo(s.Console)

		if s.path == "" {
			continue
		}

		s.setHealthy()

		f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			s.handleError(err, "failed to open log file")
			continue
		}

		stat, err := f.Stat()
		s.handleError(err, "failed to get log file status")
		beforeSize := stat.Size()

		reader.Seek(0, io.SeekStart)
		wroteSize, err := reader.WriteTo(f)
		s.handleError(err, "failed to write log file")

		err = f.Close()
		s.handleError(err, "failed to close log file")

		if s.index.AppendEntry(beforeSize, beforeSize+wroteSize, r.Time.Unix()) == ErrLogUnmatch {
			s.index.Reset()
			s.index.AppendEntry(0, beforeSize+wroteSize, r.Time.Unix())
		}
	}

	close(stopped)
}

func (s *Store) Close() error {
	close(s.writeCh)
	<-s.writerStopped
	return nil
}

// ProbeHistory returns a slice of lib-ayd.ProbeHistory.
// This method only returns active target's ProbeHistory.
func (s *Store) ProbeHistory() []api.ProbeHistory {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	var result []api.ProbeHistory
	for _, x := range s.probeHistory {
		if x.isActive() {
			result = append(result, x.MakeReport(PROBE_HISTORY_LEN))
		}
	}

	api.SortProbeHistories(result)

	return result
}

// Targets returns target URLs as a string slice.
// The result is includes inactive target, sorted in dictionary order.
func (s *Store) Targets() []string {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	result := make([]string, len(s.probeHistory))
	i := 0
	for _, x := range s.probeHistory {
		result[i] = x.Target.String()
		i++
	}

	sort.Strings(result)

	return result
}

func (s *Store) currentIncidentsWithoutLock() []*api.Incident {
	result := make([]*api.Incident, 0, len(s.currentIncidents))

	for _, x := range s.currentIncidents {
		if s.probeHistory.isActive(x.Target) {
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
		if s.probeHistory.isActive(x.Target) {
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

func (s *Store) searchLastIncident(target string, t time.Time) *api.Incident {
	cur, ok := s.currentIncidents[target]
	if ok {
		return cur
	}

	hs, hok := s.probeHistory[target]

	if hok && len(hs.Records) > 0 && hs.Records[len(hs.Records)-1].Time.Before(t) {
		return nil
	}

	for i := len(s.incidentHistory) - 1; i >= 0; i-- {
		x := s.incidentHistory[i]

		if x.Target.String() == target && t.Before(x.EndsAt) {
			if x.StartsAt.Before(t) {
				return x
			}

			if hok {
				for i := len(hs.Records) - 1; i >= 0; i-- {
					h := hs.Records[i]
					if h.Time.Before(x.StartsAt) {
						if h.Time.Before(t) {
							return x
						}
						break
					}
				}
			}
		}
	}

	return nil
}

func (s *Store) setIncidentIfNeed(r api.Record, needCallback bool) {
	if r.Status == api.StatusAborted {
		return
	}

	target := r.Target.String()

	if incident := s.searchLastIncident(target, r.Time); incident != nil {
		if incident.StartsAt.After(r.Time) {
			incident.StartsAt = r.Time
		}

		// nothing to do for continue of current incident, or for old resolved incident.
		if incident.Status == r.Status && incident.Message == r.Message && (incident.EndsAt.IsZero() || incident.EndsAt.After(r.Time)) {
			return
		}

		incident.EndsAt = r.Time
		s.incidentHistory = append(s.incidentHistory, incident)
		delete(s.currentIncidents, target)

		if len(s.incidentHistory) > INCIDENT_HISTORY_LEN {
			s.incidentHistory = s.incidentHistory[1:]
		}

		// kick incident callback when recover
		if r.Status == api.StatusHealthy && needCallback {
			for _, cb := range s.OnStatusChanged {
				cb(r)
			}
		}
	}

	if r.Status != api.StatusHealthy {
		incident := newIncident(r)

		if hs, ok := s.probeHistory[target]; ok && len(hs.Records) > 0 && hs.Records[len(hs.Records)-1].Time.After(r.Time) {
			var next api.Record

			for _, h := range hs.Records {
				if r.Time.Before(h.Time) {
					incident.EndsAt = h.Time
					next = h
					break
				}
			}
			s.incidentHistory = append(s.incidentHistory, incident)

			// kick incident callback when new incident caused
			if needCallback {
				s.incidentCount++
				for _, cb := range s.OnStatusChanged {
					cb(r)
					if next.Status == api.StatusHealthy {
						cb(next)
					}
				}
			}
		} else {
			s.currentIncidents[target] = incident

			// kick incident callback when new incident caused
			if needCallback {
				s.incidentCount++
				for _, cb := range s.OnStatusChanged {
					cb(r)
				}
			}
		}
	}
}

// Report reports a Record to this Store.
//
// See also probeHistoryMap.Append about the arguments.
func (s *Store) Report(source *api.URL, r api.Record) {
	if _, ok := r.Target.User.Password(); ok {
		r.Target.User = url.UserPassword(r.Target.User.Username(), "xxxxx")
	}
	r.Message = strings.Trim(r.Message, "\r\n")

	s.writeCh <- r

	if r.Target.Scheme != "alert" && r.Target.Scheme != "ayd" {
		s.historyLock.Lock()
		defer s.historyLock.Unlock()

		s.setIncidentIfNeed(r, true)
		s.probeHistory.Append(source, r)
	}
}

func (s *Store) Restore() error {
	if s.path == "" {
		return nil
	}

	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	s.index.Lock()
	defer s.index.Unlock()

	f, err := os.OpenFile(s.path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if ret, _ := f.Seek(-LogRestoreBytes, os.SEEK_END); ret != 0 {
		u := &api.URL{Scheme: "ayd", Opaque: "log"}
		fmt.Fprintln(s.Console, api.Record{
			Time:    time.Now(),
			Status:  api.StatusDegrade,
			Target:  u,
			Message: "WARNING: read only last 1GB from log file because it is too large",
			Extra: map[string]interface{}{
				"log_size": ret + LogRestoreBytes,
			},
		})
	} else if info, err := f.Stat(); err == nil && info.Size() > 10*1024*1024 {
		u := &api.URL{Scheme: "ayd", Opaque: "log"}
		fmt.Fprintln(s.Console, api.Record{
			Time:    time.Now(),
			Status:  api.StatusHealthy,
			Target:  u,
			Message: "WARNING: loading large log file",
			Extra: map[string]interface{}{
				"log_size": info.Size(),
			},
		})
	}

	s.probeHistory = make(probeHistoryMap)
	s.index.ResetWithoutLock()
	var fileIndex int64

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		l := int64(len(line))
		fileIndex += l

		var r api.Record
		if err = r.UnmarshalJSON(line); err != nil {
			s.index.AppendInvalidRangeWithoutLock(fileIndex-l, fileIndex)
			continue
		}

		s.index.AppendEntryWithoutLock(fileIndex-l, fileIndex, r.Time.Unix())

		if _, ok := r.Target.User.Password(); ok {
			r.Target.User = url.UserPassword(r.Target.User.Username(), "xxxxx")
		}

		if r.Target.Scheme != "alert" && r.Target.Scheme != "ayd" {
			s.setIncidentIfNeed(r, false)
			s.probeHistory.Append(r.Target, r)
		}
	}

	for k := range s.probeHistory {
		s.probeHistory[k].setInactive()
	}

	return nil
}

// ActivateTarget marks the target will reported via specified source.
// This method prepares a probeHistory, and mark it as active.
func (s *Store) ActivateTarget(source, target *api.URL) {
	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	t := target.String()

	if _, ok := s.probeHistory[t]; !ok {
		s.probeHistory[t] = &probeHistory{
			Target: target,
		}
	}

	s.probeHistory[t].addSource(source)
}

// DeactivateTarget marks the target is no longer reported via specified source.
func (s *Store) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	for _, t := range targets {
		if x, ok := s.probeHistory[t.String()]; ok {
			x.removeSource(source)
		}
	}
}

// setHealthy is reset healthy status of this store.
// This status is reported by Errors method.
func (s *Store) setHealthy() {
	s.errorsLock.Lock()
	defer s.errorsLock.Unlock()

	s.healthy = true
}

// addError adds error message for Errors method, and set healthy status to false.
// This errors reported by Errors method.
func (s *Store) addError(message string) {
	s.errorsLock.Lock()
	defer s.errorsLock.Unlock()

	s.healthy = false
	s.errors = append(
		s.errors,
		fmt.Sprintf("%s\t%s", time.Now().Format(time.RFC3339), message),
	)

	if len(s.errors) > 10 {
		s.errors = s.errors[1:]
	}
}

// Errors returns store status and error logs.
func (s *Store) Errors() (healthy bool, messages []string) {
	s.errorsLock.RLock()
	defer s.errorsLock.RUnlock()

	return s.healthy, s.errors
}

// MakeReport creates ayd.Report for exporting for endpoint.
// The result includes only information about active targets.
func (s *Store) MakeReport(probeHistoryLength int) api.Report {
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
		if v.isActive() {
			report.ProbeHistory[k] = v.MakeReport(probeHistoryLength)
		}
	}

	return report
}

// SetIndexInterval sets indexing interval for debug.
func (s *Store) SetIndexInterval(interval int64) {
	s.index.interval = interval
}
