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
	INCIDENT_HISTORY_LEN = 20
)

var (
	LogRestoreBytes int64 = 100 * 1024 * 1024
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

		reader.Seek(0, io.SeekStart)
		_, err = reader.WriteTo(f)
		s.handleError(err, "failed to write log file")

		err = f.Close()
		s.handleError(err, "failed to close log file")
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

func (s *Store) setIncidentIfNeed(r api.Record, needCallback bool) {
	if r.Status == api.StatusAborted {
		return
	}

	target := r.Target.String()
	if cur, ok := s.currentIncidents[target]; ok {
		if incidentIsContinued(cur, r) {
			return
		}

		cur.EndsAt = r.Time
		s.incidentHistory = append(s.incidentHistory, cur)
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

		s.probeHistory.Append(source, r)
		s.setIncidentIfNeed(r, true)
	}
}

func (s *Store) Restore() error {
	if s.path == "" {
		return nil
	}

	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	f, err := os.OpenFile(s.path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if ret, _ := f.Seek(-LogRestoreBytes, os.SEEK_END); ret != 0 {
		u := &api.URL{Scheme: "ayd", Opaque: "log"}
		s.Report(u, api.Record{
			Time:    time.Now(),
			Status:  api.StatusDegrade,
			Target:  u,
			Message: "WARNING: read only last 100MB from log file because it is too large",
			Extra: map[string]interface{}{
				"log_size": ret + LogRestoreBytes,
			},
		})
	}

	s.probeHistory = make(probeHistoryMap)

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
			s.probeHistory.Append(r.Target, r)
			s.setIncidentIfNeed(r, false)
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
