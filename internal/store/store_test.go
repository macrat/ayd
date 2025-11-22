package store_test

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/store"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

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

	buf := NewBuffer()
	s, err := store.New("", f.Name(), buf)
	if err != nil {
		t.Errorf("failed to open store %s (with permission 600): %s", f.Name(), err)
	}
	defer s.Close()

	s.Report(&api.URL{Scheme: "dummy"}, api.Record{
		Time:    time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
		Status:  api.StatusHealthy,
		Latency: 42 * time.Millisecond,
		Target:  &api.URL{Scheme: "dummy", Fragment: "logging-test"},
		Message: "hello world",
	})

	time.Sleep(10 * time.Millisecond)

	if buf.Line(0) != `{"time":"2001-02-03T16:05:06Z", "status":"HEALTHY", "latency":42.000, "target":"dummy:#logging-test", "message":"hello world"}` {
		t.Errorf("unexpected log (line 0):\n%s", buf)
	}

	if healthy, messages := s.Errors(); !healthy {
		t.Errorf("unexpected error recorded: %#v", messages)
	}

	os.Chmod(f.Name(), 0000)

	s.Report(&api.URL{Scheme: "dummy"}, api.Record{
		Time:    time.Date(2001, 2, 3, 16, 5, 7, 0, time.UTC),
		Status:  api.StatusHealthy,
		Latency: 42 * time.Millisecond,
		Target:  &api.URL{Scheme: "dummy", Fragment: "logging-test"},
		Message: "foo bar",
	})

	time.Sleep(10 * time.Millisecond)

	if buf.Line(-2) != `{"time":"2001-02-03T16:05:07Z", "status":"HEALTHY", "latency":42.000, "target":"dummy:#logging-test", "message":"foo bar"}` {
		t.Errorf("unexpected log (line -2):\n%s", buf)
	}

	if ok, err := regexp.MatchString(`^{"time":"[-+:TZ0-9]+", "status":"FAILURE", "latency":0.000, "target":"ayd:log", "message":"[^"]+"}$`, buf.Line(-1)); err != nil {
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
	path := filepath.Join(t.TempDir(), "%H.log")

	s1, err := store.New("", path, io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s1.Close()

	records := []api.Record{
		{
			Time:    time.Now().Add(-time.Hour),
			Target:  &api.URL{Scheme: "ping", Opaque: "restore-test"},
			Status:  api.StatusUnknown,
			Message: "hello world",
			Latency: 1 * time.Second,
		},
		{
			Time:    time.Now().Add(-30 * time.Minute),
			Target:  &api.URL{Scheme: "exec", Opaque: "/usr/local/bin/test.sh"},
			Status:  api.StatusHealthy,
			Message: "foobar",
			Latency: 123 * time.Millisecond,
		},
		{
			Time:    time.Now().Add(-15 * time.Minute),
			Target:  &api.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:  api.StatusFailure,
			Message: "hoge",
			Latency: 123 * time.Microsecond,
		},
		{
			Time:    time.Now(),
			Target:  &api.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:  api.StatusHealthy,
			Message: "hoge",
			Latency: 123 * time.Microsecond,
		},
	}

	for _, r := range records {
		s1.Report(&api.URL{Scheme: "dummy"}, r)
	}

	time.Sleep(100 * time.Millisecond) // wait for write

	s2, err := store.New("", path, io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s2.Close()

	if targets := s2.Targets(); len(targets) != 0 {
		t.Fatalf("expected no targets but got: %v", targets)
	}

	if err = s2.Restore(); err != nil {
		t.Fatalf("failed to restore store: %s", err)
	}

	if targets := s2.Targets(); len(targets) != 3 {
		t.Fatalf("expected 3 targets but got: %v", targets)
	} else if diff := cmp.Diff([]string{"exec:/usr/local/bin/test.sh", "http://test.local/abc/def", "ping:restore-test"}, targets); diff != "" {
		t.Fatalf("unexpected targets\n%s", diff)
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
			same := (x.Time == y.Time &&
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
	s, err := store.New("", "./testdata/with-password.log", io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	if err = s.Restore(); err != nil {
		t.Fatalf("failed to restore: %s", err)
	}

	u := &api.URL{Scheme: "http", User: url.UserPassword("hoge", "xxxxx"), Host: "example.com"}
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
	s, err := store.New("", "", io.Discard)
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

	s, err := store.New("", f.Name(), io.Discard)
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
		[]byte(`{"time":"2001-02-03T01:02:03Z", "status":"HEALTHY", "latency":0.123, "target":"dummy:healthy", "message":"should be ignore because this is on the border\n"}`+"\n"),
		[]byte(`{"time":"2001-02-03T02:03:04Z", "status":"HEALTHY", "latency":0.123, "target":"dummy:healthy", "message":"first record"}`+"\n")...,
	)
	f.Write(rawRecord)

	f.Write(bytes.Repeat([]byte("X"), 1000-len(rawRecord)+1)) // padding for drop first line
	store.LogRestoreBytes = 1000

	f.Sync()

	s, err := store.New("", f.Name(), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	if err := s.Restore(); err != nil {
		t.Fatalf("failed to restore log: %s", err)
	}

	u := &api.URL{Scheme: "dummy", Opaque: "healthy"}
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

func TestStore_Restore_fileRemoved(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	s, err := store.New("", f.Name(), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	baseTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 2; i++ {
		since := baseTime.Add(time.Duration(i) * 10 * time.Minute)

		err := os.Truncate(f.Name(), 0)
		if err != nil {
			t.Fatalf("%d: failed to truncate log file: %s", i, err)
		}

		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    since,
			Target:  &api.URL{Scheme: "dummy"},
			Message: "hello world",
		})
		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    since.Add(1 * time.Minute),
			Target:  &api.URL{Scheme: "dummy"},
			Message: "hello world",
		})
		time.Sleep(10 * time.Millisecond) // wait for writing goroutine

		scanner, err := s.OpenLog(since.Add(-1*time.Minute), since.Add(5*time.Minute))
		if err != nil {
			t.Fatalf("failed to open log: %s", err)
		}

		var rs []api.Record
		for scanner.Scan() {
			rs = append(rs, scanner.Record())
		}
		if len(rs) != 2 {
			t.Fatalf("unexpected number of records found:\n%#v", rs)
		}
	}
}

func TestStore_AddTarget(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	if len(s.ProbeHistory()) != 0 {
		t.Fatalf("found unexpected probe history")
	}

	s.Report(&api.URL{Scheme: "dummy"}, api.Record{
		Target:  &api.URL{Scheme: "dummy", Fragment: "add-target-1"},
		Message: "already exists history",
		Status:  api.StatusHealthy,
	})
	if len(s.ProbeHistory()) != 1 {
		t.Fatalf("found unexpected probe history")
	}

	s.ActivateTarget(&api.URL{Scheme: "dummy"}, &api.URL{Scheme: "dummy", Fragment: "add-target-2"})
	s.ActivateTarget(&api.URL{Scheme: "dummy"}, &api.URL{Scheme: "dummy", Fragment: "add-target-1"})
	s.ActivateTarget(&api.URL{Scheme: "dummy"}, &api.URL{Scheme: "dummy", Fragment: "add-target-2"})

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

		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    time.Now().Add(offset),
			Target:  &api.URL{Scheme: "dummy", User: url.UserPassword("test", password), Path: path},
			Message: message,
			Status:  status,
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

	// start another incident
	appendRecord("incident-test-3", "l", "this message (localhost:1234) includes address.", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-3")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-2")
	assertLastIncident("this message (localhost:1234) includes address.")
	assertCallbackCount(7)

	// port number difference should be ignored
	appendRecord("incident-test-3", "m", "this message (localhost:2345) includes address.", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-3")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-2")
	assertLastIncident("this message (localhost:1234) includes address.")
	if s.CurrentIncidents()[0].Message != "this message (localhost:2345) includes address." {
		t.Errorf("incident message should be updated: %s", s.CurrentIncidents()[0].Message)
	}
	assertCallbackCount(7)

	// well-known port number should not be ignored
	appendRecord("incident-test-3", "n", "this message (localhost:80) includes address.", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy://test:xxxxx@incident-test-3")
	assertIncidents(s.IncidentHistory(), "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-1", "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-2", "dummy://test:xxxxx@incident-test-3")
	assertLastIncident("this message (localhost:80) includes address.")
	assertCallbackCount(8)
}

func TestStore_delayedIncident(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	var messages []string
	callbackCount := 0
	s.OnStatusChanged = []store.RecordHandler{
		func(s *[]string, c *int) func(api.Record) {
			return func(r api.Record) {
				*s = append(*s, r.Message)
				*c++
			}
		}(&messages, &callbackCount),
	}

	assert := func(count int, from, to int64, message ...string) {
		t.Helper()

		if count != callbackCount {
			t.Fatalf("unexpected number of callbacks: expected %d but found %d", count, callbackCount)
		}
		for i := 0; i < len(messages); i++ {
			if message[i] != messages[i] {
				t.Fatalf("unexpected incident message[%d]: expected %q but found %q", i, message[i], messages[i])
			}
		}

		is := s.CurrentIncidents()
		if len(is) != 0 {
			t.Fatalf("unexpected current incidents found: %v", is)
		}

		is = s.IncidentHistory()
		if len(is) == 0 {
			t.Fatalf("incident not found")
		} else {
			i := is[len(is)-1]

			if i.Target.String() != "dummy:" {
				t.Fatalf("unexpected incident found: %s", i)
			}

			if i.StartsAt.Unix() != from {
				t.Fatalf("incident should begins at %d but begins at %d", from, i.StartsAt.Unix())
			}
			if i.EndsAt.Unix() != to {
				t.Errorf("incident should ends at %d but ends at %d", to, i.EndsAt.Unix())
			}
			if t.Failed() {
				t.FailNow()
			}
		}
	}
	report := func(offset int64, message string, status api.Status) {
		t.Helper()

		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    time.Unix(0, 0).Add(time.Duration(offset) * time.Second),
			Target:  &api.URL{Scheme: "dummy"},
			Message: message,
			Status:  status,
		})
	}

	// offset  05  10  15  20  25  30  35  40  50
	// status   F   H   F   F   F   F   F   H   H
	// order    |   1   |   2   |   3   |   4   5
	//          |       |       6       |          -- First test. Put into middle.
	//          |       |               7          -- Second test. Put into very end.
	//          |       8                          -- Third test. Put into before begin.
	//          9                                  -- Fourth test. New incident.

	report(10, "hello1", api.StatusHealthy)
	report(20, "oh no", api.StatusFailure)
	report(30, "oh no", api.StatusFailure)
	report(40, "hello2", api.StatusHealthy)
	report(50, "hello3", api.StatusHealthy)
	assert(2, 20, 40, "oh no", "hello2")

	report(25, "oh no", api.StatusFailure) // First test
	assert(2, 20, 40, "oh no", "hello2")

	report(35, "oh no", api.StatusFailure) // Second test
	assert(2, 20, 40, "oh no", "hello2")

	report(15, "oh no", api.StatusFailure) // Third test
	assert(2, 15, 40, "oh no", "hello2")

	report(5, "wah", api.StatusFailure) // Fourth test
	assert(4, 5, 10, "oh no", "hello2", "wah", "hello1")
}

func TestStore_incident_len_limit(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	for i := 0; i < store.INCIDENT_HISTORY_LEN*2; i++ {
		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Target:  &api.URL{Scheme: "dummy", Fragment: "history-limit-test"},
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

	s1, err := store.New("", "", io.Discard)
	if err != nil {
		t.Fatalf("failed to create store")
	}
	defer s1.Close()

	s1.Report(&api.URL{Scheme: "dummy"}, api.Record{
		Target:  &api.URL{Scheme: "dummy", Fragment: "empty-path"},
		Message: "hello world",
		Status:  api.StatusHealthy,
	})
	if len(s1.ProbeHistory()) != 1 {
		t.Errorf("unexpected number of probe history: %d", len(s1.ProbeHistory()))
	}

	time.Sleep(10 * time.Millisecond) // waiting for writer

	s2, err := store.New("", "", io.Discard)
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

	s, err := store.New("", "", buf)
	if err != nil {
		t.Fatalf("failed to create store")
	}

	s.ReportInternalError("test", "hello world")

	time.Sleep(10 * time.Millisecond) // wait for writer

	s.Close()

	time.Sleep(10 * time.Millisecond) // wait for close

	if ok, err := regexp.MatchString(`^{"time":"[-+:ZT0-9]+", "status":"FAILURE", "latency":[.0-9]+, "target":"ayd:test", "message":"hello world"}`+"\n$", buf.String()); err != nil {
		t.Fatalf("failed to match log: %s", err)
	} else if !ok {
		t.Errorf("unexpected error:\n%s", buf.String())
	}
}

func TestStore_logRotate(t *testing.T) {
	dir := t.TempDir()

	s, err := store.New("", filepath.Join(dir, "dt=%Y%m%d/%H.log"), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}

	report := func(Y, m, d, H, M int) {
		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Time:   time.Date(Y, time.Month(m), d, H, M, 0, 0, time.UTC),
			Target: &api.URL{Scheme: "dummy"},
		})
	}
	assert := func(lines ...int) {
		t.Helper()
		ls := s.Pathes()
		if len(ls) != len(lines) {
			t.Fatalf("unexpected number of log files found: expected=%d got=%d", len(lines), len(ls))
		}
		for i, f := range ls {
			bs, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("%d: failed to read %s: %s", i, f, err)
			}
			if n := bytes.Count(bs, []byte{'\n'}); n != lines[i] {
				t.Fatalf("%d: unexpected number of lines found in %s: expected=%d got=%d", i, f, lines[i], n)
			}
		}
	}

	wait := 10 * time.Millisecond
	if runtime.GOOS == "windows" {
		wait = 100 * time.Millisecond
	}

	assert()

	report(2001, 2, 3, 16, 5)
	report(2001, 2, 3, 16, 6)
	time.Sleep(wait)
	assert(2)

	report(2001, 2, 4, 16, 50)
	report(2001, 2, 3, 16, 7)
	time.Sleep(wait)
	assert(3, 1)

	report(2001, 2, 4, 16, 5)
	report(2001, 2, 3, 4, 5)
	time.Sleep(wait)
	assert(1, 3, 2)

	r, err := s.OpenLog(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to open scanner: %s", err)
	}
	defer r.Close()

	tests := []struct {
		Day, Hour, Minute int
	}{
		{3, 4, 5},
		{3, 16, 5},
		{3, 16, 6},
		{3, 16, 7},
		{4, 16, 50},
		{4, 16, 5},
	}
	for i, tt := range tests {
		if !r.Scan() {
			t.Fatalf("%d: failed to scan", i)
		}

		got := r.Record().Time

		if got.Day() != tt.Day || got.Hour() != tt.Hour || got.Minute() != tt.Minute {
			t.Errorf("%d: unexpected date record found: want=02/%02d %02d:%02d actual=02/%02d %02d:%02d", i, tt.Day, tt.Hour, tt.Minute, got.Day(), got.Hour(), got.Minute())
		}
	}

	if r.Scan() {
		t.Fatalf("unexpected extra record found: %s", r.Record())
	}
}

func TestStore_MakeReport(t *testing.T) {
	s := testutil.NewStore(t, testutil.WithLog())
	defer s.Close()

	assert := func(targetCount, currentIncidentCount, incidentHistoryCount int) {
		t.Helper()

		r := s.MakeReport(store.PROBE_HISTORY_LEN)

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

		s.Report(&api.URL{Scheme: "dummy"}, api.Record{
			Time:   timestamp,
			Status: status,
			Target: &api.URL{Scheme: "dummy", Fragment: fragment},
		})
	}

	assert(1, 0, 0)

	addLog("1", api.StatusHealthy)
	assert(2, 0, 0)

	addLog("1", api.StatusHealthy)
	assert(2, 0, 0)

	addLog("2", api.StatusHealthy)
	assert(3, 0, 0)

	addLog("1", api.StatusFailure)
	assert(3, 1, 0)

	addLog("1", api.StatusHealthy)
	assert(3, 0, 1)
}

func TestStore_Name(t *testing.T) {
	withoutName, err := store.New("", filepath.Join(t.TempDir(), "ayd.log"), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	if withoutName.Name() != "" {
		t.Errorf("unexpected name for store without name: %q", withoutName.Name())
	}
	if n := withoutName.MakeReport(10).InstanceName; n != "" {
		t.Errorf("unexpected name in report for store without name: %q", n)
	}

	withName, err := store.New("test", filepath.Join(t.TempDir(), "ayd.log"), io.Discard)
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	if withName.Name() != "test" {
		t.Errorf("unexpected name for store with name: %q", withName.Name())
	}
	if n := withName.MakeReport(10).InstanceName; n != "test" {
		t.Errorf("unexpected name in report for store with name: %q", n)
	}
}

func BenchmarkStore_Append(b *testing.B) {
	for _, status := range []api.Status{api.StatusHealthy, api.StatusFailure} {
		b.Run(status.String(), func(b *testing.B) {
			s := testutil.NewStore(b)
			defer s.Close()

			source := &api.URL{Scheme: "dummy"}

			record := api.Record{
				Time:    time.Now(),
				Target:  &api.URL{Scheme: "dummy", Fragment: "benchmark-append"},
				Status:  status,
				Message: "hello world",
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
		s.MakeReport(store.PROBE_HISTORY_LEN)
	}
}
