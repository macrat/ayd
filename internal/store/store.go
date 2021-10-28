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
	sources []string
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

// addSource appends reporter URL that reports to this ProbeHistory.
// The sources will used to detect if is this target active or not.
func (ph *ProbeHistory) addSource(source *url.URL) {
	s := source.Redacted()
	for _, x := range ph.sources {
		if x == s {
			return
		}
	}

	ph.sources = append(ph.sources, s)
}

// removeSource removes reporter URL, that reports to this ProbeHistory, from sources.
func (ph *ProbeHistory) removeSource(source *url.URL) {
	s := source.Redacted()
	for i, x := range ph.sources {
		if x == s {
			ph.sources = append(ph.sources[:i], ph.sources[i+1:]...)
			return
		}
	}
}

// setInactive removes all reporter URLs from this ProbeHistory.
func (ph *ProbeHistory) setInactive() {
	ph.sources = nil
}

// isActive returns if is this ProbeHistory active in current execution or not.
func (ph ProbeHistory) isActive() bool {
	return len(ph.sources) != 0
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

// Append adds ayd.Record to the ProbeHistory.
//
// `source` of argument means who is reporting this record.
// In the almost cases, it is the same as r.Target, but some cases like `source:` have another URL.
func (hs ProbeHistoryMap) Append(source *url.URL, r api.Record) {
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

	hs[target].addSource(source)
}

// isActive returns if is the specified target active in current execution or not.
func (hs ProbeHistoryMap) isActive(target *url.URL) bool {
	return hs[target.Redacted()].isActive()
}

type RecordHandler func(api.Record)

type Store struct {
	Path string

	Console io.Writer

	historyLock      sync.RWMutex
	probeHistory     ProbeHistoryMap
	currentIncidents map[string]*api.Incident
	incidentHistory  []*api.Incident

	OnStatusChanged []RecordHandler
	IncidentCount   int

	writeCh       chan<- api.Record
	writerStopped chan struct{}
	errorsLock    sync.RWMutex
	errors        []string
	healthy       bool
}

func New(path string, console io.Writer) (*Store, error) {
	ch := make(chan api.Record, 32)

	store := &Store{
		Path:             path,
		Console:          console,
		probeHistory:     make(ProbeHistoryMap),
		currentIncidents: make(map[string]*api.Incident),
		writeCh:          ch,
		writerStopped:    make(chan struct{}),
		healthy:          true,
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
	s.Report(&url.URL{Scheme: "ayd", Opaque: scope}, api.Record{
		CheckedAt: time.Now(),
		Status:    api.StatusFailure,
		Target:    &url.URL{Scheme: "ayd", Opaque: scope},
		Message:   message,
	})
}

// handleError reports an error of write a log.
// This error will reported to console in this method, and /healthz page via Store.Errors method.
func (s *Store) handleError(err error, exportableErrorMessage string) {
	if err != nil {
		s.addError(exportableErrorMessage)
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

		s.setHealthy()

		f, err := os.OpenFile(s.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			s.handleError(err, "failed to open log file")
			continue
		}

		_, err = f.Write(msg)
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

// ProbeHistory returns a slice of ProbeHistory.
// This method only returns active target's ProbeHistory.
func (s *Store) ProbeHistory() []*ProbeHistory {
	s.historyLock.RLock()
	defer s.historyLock.RUnlock()

	var result []*ProbeHistory
	for _, x := range s.probeHistory {
		if x.isActive() {
			result = append(result, x)
		}
	}

	sort.Sort(byLatestStatus(result))

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

		// kick incident callback when recover
		if r.Status == api.StatusHealthy && needCallback {
			for _, cb := range s.OnStatusChanged {
				cb(r)
			}
		}
	}

	if r.Status != api.StatusHealthy {
		incident := NewIncident(r)
		s.currentIncidents[target] = incident

		// kick incident callback when new incident caused
		if needCallback {
			s.IncidentCount++
			for _, cb := range s.OnStatusChanged {
				cb(r)
			}
		}
	}
}

// Report reports a Record to this Store.
//
// See also ProbeHistoryMap.Append about the arguments.
func (s *Store) Report(source *url.URL, r api.Record) {
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
// This method prepares a ProbeHistory, and mark it as active.
func (s *Store) ActivateTarget(source, target *url.URL) {
	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	t := target.Redacted()

	if _, ok := s.probeHistory[t]; !ok {
		s.probeHistory[t] = &ProbeHistory{
			Target: target,
		}
	}

	s.probeHistory[t].addSource(source)
}

// DeactivateTarget marks the target is no longer reported via specified source.
func (s *Store) DeactivateTarget(source *url.URL, targets ...*url.URL) {
	s.historyLock.Lock()
	defer s.historyLock.Unlock()

	for _, t := range targets {
		if x, ok := s.probeHistory[t.Redacted()]; ok {
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
		if v.isActive() {
			report.ProbeHistory[k] = v.MakeReport()
		}
	}

	return report
}
