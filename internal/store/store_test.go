package store_test

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/store"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestProbeHistoryMap(t *testing.T) {
	m := make(store.ProbeHistoryMap)

	for i := 1; i <= 100; i++ {
		m.Append(&url.URL{Scheme: "dummy"}, api.Record{
			CheckedAt: time.Now().Add(time.Duration(i) * time.Second),
			Target:    &url.URL{Scheme: "dummy", Fragment: "append-test"},
			Message:   fmt.Sprint(i),
		})
	}

	if hs, ok := m["dummy:#append-test"]; !ok {
		t.Errorf("failed to get history\n%#v", m)
	} else if len(hs.Records) != store.PROBE_HISTORY_LEN {
		t.Errorf("unexpected number of records: %d", len(hs.Records))
	} else if hs.Records[len(hs.Records)-1].Message != "100" {
		t.Errorf("unexpected message of latest record: %#v", hs.Records[len(hs.Records)-1])
	}

	for i := 1; i <= 10; i++ {
		m.Append(&url.URL{Scheme: "dummy"}, api.Record{
			CheckedAt: time.Now().Add(time.Duration(i) * time.Second),
			Target:    &url.URL{Scheme: "dummy", Fragment: "append-test-another"},
			Message:   fmt.Sprint(i),
		})
	}

	if hs, ok := m["dummy:#append-test-another"]; !ok {
		t.Errorf("failed to get history\n%#v", m)
	} else if len(hs.Records) != 10 {
		t.Errorf("unexpected number of records: %d", len(hs.Records))
	} else if hs.Records[len(hs.Records)-1].Message != "10" {
		t.Errorf("unexpected message of latest record: %#v", hs.Records[len(hs.Records)-1])
	}

	for i := 1; i <= 10; i++ {
		m.Append(&url.URL{Scheme: "dummy"}, api.Record{
			CheckedAt: time.Now().Add(time.Duration(-i) * time.Second),
			Target:    &url.URL{Scheme: "dummy", Fragment: "append-test-reverse"},
			Message:   fmt.Sprint(i),
		})
	}

	if hs, ok := m["dummy:#append-test-reverse"]; !ok {
		t.Errorf("failed to get history\n%#v", m)
	} else if len(hs.Records) != 10 {
		t.Errorf("unexpected number of records: %d", len(hs.Records))
	} else if hs.Records[len(hs.Records)-1].Message != "1" {
		t.Errorf("unexpected message of latest record: %#v", hs.Records[len(hs.Records)-1])
	}

	timestamp := time.Now()
	m.Append(&url.URL{Scheme: "dummy"}, api.Record{
		CheckedAt: timestamp,
		Target:    &url.URL{Scheme: "dummy", Fragment: "append-test-same-time"},
		Message:   "first",
	})
	m.Append(&url.URL{Scheme: "dummy"}, api.Record{
		CheckedAt: timestamp,
		Target:    &url.URL{Scheme: "dummy", Fragment: "append-test-same-time"},
		Message:   "second",
	})

	if hs, ok := m["dummy:#append-test-same-time"]; !ok {
		t.Errorf("failed to get history\n%#v", m)
	} else if len(hs.Records) != 2 {
		t.Errorf("unexpected number of records: %d", len(hs.Records))
	} else if hs.Records[len(hs.Records)-1].Message != "second" {
		t.Errorf("unexpected message of latest record: %#v", hs.Records[len(hs.Records)-1])
	}
}

type Buffer struct {
	sync.Mutex

	buf *bytes.Buffer
}

func NewBuffer() *Buffer {
	return &Buffer{
		buf: &bytes.Buffer{},
	}
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.buf.Write(p)
}

func (b *Buffer) String() string {
	b.Lock()
	defer b.Unlock()
	return b.buf.String()
}

func (b *Buffer) Line(n int) string {
	xs := strings.Split(strings.Trim(b.String(), "\r\n"), "\n")
	return xs[(len(xs)+n)%len(xs)]
}

func TestStore_errorLogging(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("can't do this test because file permission does not work on windows")
		return
	}

	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	os.Chmod(f.Name(), 0000)

	_, err = store.New(f.Name(), io.Discard)
	if err == nil {
		t.Errorf("expected failed to open %s (with permission 000) but successed", f.Name())
	}

	os.Chmod(f.Name(), 0600)

	buf := NewBuffer()
	s, err := store.New(f.Name(), buf)
	if err != nil {
		t.Errorf("failed to open store %s (with permission 600)", err)
	}
	defer s.Close()

	s.Report(&url.URL{Scheme: "dummy"}, api.Record{
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
		Status:    api.StatusHealthy,
		Latency:   42 * time.Millisecond,
		Target:    &url.URL{Scheme: "dummy", Fragment: "logging-test"},
		Message:   "hello world",
	})

	time.Sleep(10 * time.Millisecond)

	if buf.Line(0) != "2001-02-03T16:05:06Z\tHEALTHY\t42.000\tdummy:#logging-test\thello world" {
		t.Errorf("unexpected log (line 0):\n%s", buf)
	}

	if healthy, messages := s.Errors(); !healthy {
		t.Errorf("unexpected error recorded: %#v", messages)
	}

	os.Chmod(f.Name(), 0000)

	s.Report(&url.URL{Scheme: "dummy"}, api.Record{
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 7, 0, time.UTC),
		Status:    api.StatusHealthy,
		Latency:   42 * time.Millisecond,
		Target:    &url.URL{Scheme: "dummy", Fragment: "logging-test"},
		Message:   "foo bar",
	})

	time.Sleep(10 * time.Millisecond)

	if buf.Line(-2) != "2001-02-03T16:05:07Z\tHEALTHY\t42.000\tdummy:#logging-test\tfoo bar" {
		t.Errorf("unexpected log (line -2):\n%s", buf)
	}

	if ok, err := regexp.MatchString("^[-+:TZ0-9]+\tFAILURE\t0.000\tayd:log\t[^\t]+$", buf.Line(-1)); err != nil {
		t.Errorf("failed to compare log (line -1): %s", err)
	} else if !ok {
		t.Errorf("unexpected log:\n%s", buf)
	}

	if healthy, messages := s.Errors(); healthy {
		t.Errorf("expect error recorded but not recorded")
	} else if len(messages) != 1 {
		t.Errorf("unexpected number of errors recorded: %#v", messages)
	} else if ok, _ := regexp.MatchString("^[-+:TZ0-9]+\tfailed to open log file$", messages[0]); !ok {
		t.Errorf("unexpected error message recorded: %#v", messages)
	}
}

func TestStore_Restore(t *testing.T) {
	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	s1, err := store.New(f.Name(), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s1.Close()

	records := []api.Record{
		{
			CheckedAt: time.Now().Add(-30 * time.Minute),
			Target:    &url.URL{Scheme: "ping", Opaque: "restore-test"},
			Status:    api.StatusUnknown,
			Message:   "hello world",
			Latency:   1 * time.Second,
		},
		{
			CheckedAt: time.Now().Add(-20 * time.Minute),
			Target:    &url.URL{Scheme: "exec", Opaque: "/usr/local/bin/test.sh"},
			Status:    api.StatusHealthy,
			Message:   "foobar",
			Latency:   123 * time.Millisecond,
		},
		{
			CheckedAt: time.Now().Add(-10 * time.Minute),
			Target:    &url.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:    api.StatusFailure,
			Message:   "hoge",
			Latency:   123 * time.Microsecond,
		},
		{
			CheckedAt: time.Now(),
			Target:    &url.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:    api.StatusHealthy,
			Message:   "hoge",
			Latency:   123 * time.Microsecond,
		},
	}

	for _, r := range records {
		s1.Report(&url.URL{Scheme: "dummy"}, r)
	}

	time.Sleep(100 * time.Millisecond) // wait for write

	s2, err := store.New(f.Name(), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s2.Close()

	if err = s2.Restore(); err != nil {
		t.Fatalf("failed to restore store: %s", err)
	}

	hs1 := s1.ProbeHistory()
	hs2 := s2.ProbeHistory()

	if len(hs2) != 0 {
		t.Errorf("unexpected history length: %d (histories should inactive yet)", len(hs2))
	}

	for _, x := range hs1 {
		s2.ActivateTarget(x.Target, x.Target)
	}
	hs2 = s2.ProbeHistory()

	if len(hs1) != len(hs2) {
		t.Fatalf("unexpected history length: %d", len(s2.ProbeHistory()))
	}

	for i := range hs1 {
		ph1 := hs1[i]
		ph2 := hs2[i]

		if ph1.Target.String() != ph2.Target.String() {
			t.Errorf("%d: different target %s != %s", i, ph1.Target, ph2.Target)
			continue
		}

		if len(ph1.Records) != len(ph2.Records) {
			t.Errorf("%d: unmatch restored records number: %d != %d", i, len(ph1.Records), len(ph2.Records))
			continue
		}

		for j := range ph1.Records {
			x := ph1.Records[j]
			y := ph2.Records[j]
			same := (x.CheckedAt == y.CheckedAt &&
				x.Target.String() != y.Target.String() &&
				x.Status == y.Status &&
				x.Message == y.Message &&
				x.Latency == y.Latency)
			if same {
				t.Errorf("%d %d: unexpected record", i, j)
			}
		}
	}

	for _, x := range hs1 {
		s2.DeactivateTarget(x.Target, x.Target)
	}
	hs2 = s2.ProbeHistory()

	if len(hs2) != 0 {
		t.Fatalf("deactivated but there are still %d histories", len(s2.ProbeHistory()))
	}
}

func TestStore_Restore_removePassword(t *testing.T) {
	s, err := store.New("./testdata/with-password.log", io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}

	u := &url.URL{Scheme: "http", User: url.UserPassword("hoge", "xxxxx"), Host: "example.com"}
	s.ActivateTarget(u, u)

	hs := s.ProbeHistory()

	if len(hs) != 1 {
		t.Fatalf("unexpected number of history: %d", len(hs))
	}

	if hs[0].Target.String() != "http://hoge:xxxxx@example.com" {
		t.Fatalf("unexpected target in history: %s", hs[0].Target)
	}
}

func TestStore_Restore_disableLog(t *testing.T) {
	s, err := store.New("", io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}
}

func TestStore_Restore_permission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("can't do this test because file permission does not work on windows")
		return
	}

	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	s, err := store.New(f.Name(), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	os.Chmod(f.Name(), 0000)

	err = s.Restore()
	if err == nil {
		t.Fatalf("expected error but got nil")
	} else if !os.IsPermission(err) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestStore_Restore_limitBorder(t *testing.T) {
	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	rawRecord := append(
		[]byte("2001-02-03T01:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tshould be ignore because this is on the border\n"),
		[]byte("2001-02-03T02:03:04Z\tHEALTHY\t0.123\tdummy:healthy\tfirst record\n")...,
	)
	f.Write(rawRecord)

	f.Write(bytes.Repeat([]byte("X"), 1000-len(rawRecord)+1)) // padding for drop first line
	store.LogRestoreBytes = 1000

	f.Sync()

	s, err := store.New(f.Name(), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	if err := s.Restore(); err != nil {
		t.Fatalf("failed to restore log: %s", err)
	}

	u := &url.URL{Scheme: "dummy", Opaque: "healthy"}
	s.ActivateTarget(u, u)

	hs := s.ProbeHistory()
	if len(hs) != 1 {
		t.Fatalf("unexpected number of targets: %d", len(hs))
	}

	if len(hs[0].Records) != 1 {
		t.Fatalf("unexpected number of records: %d\n%#v", len(hs[0].Records), hs[0].Records)
	}

	if hs[0].Records[0].Message != "first record" {
		t.Fatalf("unexpected first record's message: %s", hs[0].Records[0].Message)
	}
}

func TestStore_AddTarget(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	if len(s.ProbeHistory()) != 0 {
		t.Fatalf("found unexpected probe history")
	}

	s.Report(&url.URL{Scheme: "dummy"}, api.Record{
		Target:  &url.URL{Scheme: "dummy", Fragment: "add-target-1"},
		Message: "already exists history",
		Status:  api.StatusHealthy,
	})
	if len(s.ProbeHistory()) != 1 {
		t.Fatalf("found unexpected probe history")
	}

	s.ActivateTarget(&url.URL{Scheme: "dummy"}, &url.URL{Scheme: "dummy", Fragment: "add-target-2"})
	s.ActivateTarget(&url.URL{Scheme: "dummy"}, &url.URL{Scheme: "dummy", Fragment: "add-target-1"})
	s.ActivateTarget(&url.URL{Scheme: "dummy"}, &url.URL{Scheme: "dummy", Fragment: "add-target-2"})

	if len(s.ProbeHistory()) != 2 {
		t.Fatalf("unexpected length probe history: %d", len(s.ProbeHistory()))
	}

	hs := s.ProbeHistory()

	if hs[0].Target.String() != "dummy:#add-target-1" {
		t.Errorf("unexpected 1st target: %s", hs[0].Target)
	}
	if len(hs[0].Records) != 1 || hs[0].Records[0].Message != "already exists history" {
		t.Errorf("1st target's record may override: %#v", hs[0].Records)
	}

	if hs[1].Target.String() != "dummy:#add-target-2" {
		t.Errorf("unexpected 2nd target: %s", hs[1].Target)
	}
	if len(hs[1].Records) != 0 {
		t.Errorf("2nd target has unexpected record: %#v", hs[1].Records)
	}
}

func TestStore_incident(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	lastIncident := ""
	callbackCount := 0
	s.OnStatusChanged = []store.RecordHandler{
		func(s *string, c *int) func(api.Record) {
			return func(r api.Record) {
				*s = r.Message
				*c++
			}
		}(&lastIncident, &callbackCount),
	}

	assertLastIncident := func(s *string) func(string) {
		return func(expect string) {
			t.Helper()
			if *s != expect {
				t.Fatalf("expected last incident is %#v but got %#v", expect, *s)
			}
		}
	}(&lastIncident)
	assertCallbackCount := func(c *int) func(int) {
		return func(expect int) {
			t.Helper()
			if *c != expect {
				t.Fatalf("expected callback calls %d times but called %d times", expect, *c)
			}
		}
	}(&callbackCount)
	assertIncidents := func(incidents []*api.Incident, target ...string) {
		t.Helper()

		if len(incidents) != len(target) {
			ss := []string{}
			for _, i := range incidents {
				ss = append(ss, i.Target.String())
			}
			t.Fatalf("expected %d incidents but found %d incidents\n[%s]", len(target), len(incidents), strings.Join(ss, ", "))
		}

		ok := true
		for i := range target {
			if incidents[i].Target.String() != target[i] {
				t.Errorf("expected %s but got %s", target[i], incidents[i].Target)
				ok = false
			}
		}
		if !ok {
			t.FailNow()
		}
	}

	var offset time.Duration
	appendRecord := func(path, password, message string, status api.Status) {
		t.Helper()
		offset += 1 * time.Second

		s.Report(&url.URL{Scheme: "dummy"}, api.Record{
			CheckedAt: time.Now().Add(offset),
			Target:    &url.URL{Scheme: "dummy", User: url.UserPassword("test", password), Path: path},
			Message:   message,
			Status:    status,
		})
	}

	// a healthy is not an incident
	appendRecord("incident-test-1", "a", "1-1", api.StatusHealthy)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory())
	assertLastIncident("")
	assertCallbackCount(0)

	// a failure should recorded as an incident
	appendRecord("incident-test-1", "b", "1-2", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-1")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("1-2")
	assertCallbackCount(1)

	// the same incident should not recorded as a new incident
	appendRecord("incident-test-1", "c", "1-2", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-1")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("1-2")
	assertCallbackCount(1)

	// a failure for other target should recorded as a new incident
	appendRecord("incident-test-2", "d", "2-1", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("2-1")
	assertCallbackCount(2)

	// a different message should recorded as another incident
	appendRecord("incident-test-1", "e", "1-3", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1")
	assertLastIncident("1-3")
	assertCallbackCount(3)

	// the same incident should not recorded as a new incident even if there was another incident
	appendRecord("incident-test-2", "f", "2-1", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1")
	assertLastIncident("1-3")
	assertCallbackCount(3)

	// an aborted record should be ignored
	appendRecord("incident-test-2", "g", "2-?", api.StatusAborted)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1")
	assertLastIncident("1-3")
	assertCallbackCount(3)

	// a healthy should make incident mark as recovered
	appendRecord("incident-test-1", "h", "1-4", api.StatusHealthy)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-2")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1")
	assertLastIncident("1-4")
	assertCallbackCount(4)

	// a record that has different status should recorded as a new incident even if the message is the same as previous record
	appendRecord("incident-test-2", "i", "2-1", api.StatusUnknown)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-2")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2")
	assertLastIncident("2-1")
	assertCallbackCount(5)

	// a healthy should make incident mark as recovered again
	appendRecord("incident-test-2", "j", "2-2", api.StatusHealthy)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-2")
	assertLastIncident("2-2")
	assertCallbackCount(6)

	// an aborted record should be ignored again
	appendRecord("incident-test-2", "k", "2-?", api.StatusAborted)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-2")
	assertLastIncident("2-2")
	assertCallbackCount(6)
}

func TestStore_incident_len_limit(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	for i := 0; i < store.INCIDENT_HISTORY_LEN*2; i++ {
		s.Report(&url.URL{Scheme: "dummy"}, api.Record{
			Target:  &url.URL{Scheme: "dummy", Fragment: "history-limit-test"},
			Message: fmt.Sprintf("incident-%d", i),
			Status:  api.StatusFailure,
		})
	}

	if len(s.IncidentHistory()) != store.INCIDENT_HISTORY_LEN {
		t.Fatalf("unexpected incident history length: %d (expected maximum is %d)", len(s.IncidentHistory()), store.INCIDENT_HISTORY_LEN)
	}
}

func TestStore_Path_empty(t *testing.T) {
	t.Parallel()

	s1, err := store.New("", io.Discard)
	if err != nil {
		t.Fatalf("failed to create store")
	}
	defer s1.Close()

	s1.Report(&url.URL{Scheme: "dummy"}, api.Record{
		Target:  &url.URL{Scheme: "dummy", Fragment: "empty-path"},
		Message: "hello world",
		Status:  api.StatusHealthy,
	})
	if len(s1.ProbeHistory()) != 1 {
		t.Errorf("unexpected number of probe history: %d", len(s1.ProbeHistory()))
	}

	time.Sleep(10 * time.Millisecond) // waiting for writer

	s2, err := store.New("", io.Discard)
	if err != nil {
		t.Fatalf("failed to create store")
	}
	defer s2.Close()

	if len(s2.ProbeHistory()) != 0 {
		t.Errorf("unexpected number of probe history: %d", len(s2.ProbeHistory()))
	}
}

func TestStore_ReportInternalError(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}

	s, err := store.New("", buf)
	if err != nil {
		t.Fatalf("failed to create store")
	}

	s.ReportInternalError("test", "hello world")

	time.Sleep(10 * time.Millisecond) // wait for writer

	s.Close()

	time.Sleep(10 * time.Millisecond) // wait for close

	if ok, err := regexp.MatchString("^[-+:ZT0-9]+\tFAILURE\t[.0-9]+\tayd:test\thello world\n$", buf.String()); err != nil {
		t.Fatalf("failed to match log: %s", err)
	} else if !ok {
		t.Errorf("unexpected error:\n%s", buf.String())
	}
}

func TestStore_MakeReport(t *testing.T) {
	s := testutil.NewStoreWithLog(t)
	defer s.Close()

	assert := func(targetCount, currentIncidentCount, incidentHistoryCount int) {
		t.Helper()

		r := s.MakeReport()

		if len(r.ProbeHistory) != targetCount {
			t.Errorf("unexpected target count in the report: %d != %d", len(r.ProbeHistory), targetCount)
		}

		if len(r.CurrentIncidents) != currentIncidentCount {
			t.Errorf("unexpected current incident count in the report: want=%d actual=%d", currentIncidentCount, len(r.CurrentIncidents))
		}

		if len(r.IncidentHistory) != incidentHistoryCount {
			t.Errorf("unexpected incident history count in the report: want=%d actual=%d", incidentHistoryCount, len(r.IncidentHistory))
		}
	}

	timestamp := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	addLog := func(fragment string, status api.Status) {
		timestamp = timestamp.Add(time.Second)

		s.Report(&url.URL{Scheme: "dummy"}, api.Record{
			CheckedAt: timestamp,
			Status:    status,
			Target:    &url.URL{Scheme: "dummy", Fragment: fragment},
		})
	}

	assert(0, 0, 0)

	addLog("1", api.StatusHealthy)
	assert(1, 0, 0)

	addLog("1", api.StatusHealthy)
	assert(1, 0, 0)

	addLog("2", api.StatusHealthy)
	assert(2, 0, 0)

	addLog("1", api.StatusFailure)
	assert(2, 1, 0)

	addLog("1", api.StatusHealthy)
	assert(2, 0, 1)
}

func BenchmarkStore_Append(b *testing.B) {
	for _, status := range []api.Status{api.StatusHealthy, api.StatusFailure} {
		b.Run(status.String(), func(b *testing.B) {
			s := testutil.NewStore(b)
			defer s.Close()

			source := &url.URL{Scheme: "dummy"}

			record := api.Record{
				CheckedAt: time.Now(),
				Target:    &url.URL{Scheme: "dummy", Fragment: "benchmark-append"},
				Status:    status,
				Message:   "hello world",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Report(source, record)
			}
		})
	}
}

func BenchmarkStore_MakeReport(b *testing.B) {
	s := testutil.NewStore(b)
	defer s.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.MakeReport()
	}
}
