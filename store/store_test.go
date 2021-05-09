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

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestProbeHistoryMap(t *testing.T) {
	m := make(store.ProbeHistoryMap)

	for i := 1; i <= 100; i++ {
		m.Append(api.Record{
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
		m.Append(api.Record{
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
		m.Append(api.Record{
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
	m.Append(api.Record{
		CheckedAt: timestamp,
		Target:    &url.URL{Scheme: "dummy", Fragment: "append-test-same-time"},
		Message:   "first",
	})
	m.Append(api.Record{
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

func TestErrorLogging(t *testing.T) {
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

	_, err = store.New(f.Name())
	if err == nil {
		t.Errorf("expected failed to open %s (with permission 000) but successed", f.Name())
	}

	os.Chmod(f.Name(), 0600)

	s, err := store.New(f.Name())
	if err != nil {
		t.Errorf("failed to open store %s (with permission 600)", err)
	}
	buf := NewBuffer()
	s.Console = buf
	defer s.Close()

	s.Report(api.Record{
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

	os.Chmod(f.Name(), 0000)

	s.Report(api.Record{
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
}

func TestStore_restore(t *testing.T) {
	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	s1, err := store.New(f.Name())
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	s1.Console = io.Discard
	defer s1.Close()

	records := []api.Record{
		api.Record{
			CheckedAt: time.Now().Add(30 * time.Second),
			Target:    &url.URL{Scheme: "ping", Opaque: "restore-test"},
			Status:    api.StatusUnknown,
			Message:   "hello world",
			Latency:   1 * time.Second,
		},
		api.Record{
			CheckedAt: time.Now().Add(20 * time.Second),
			Target:    &url.URL{Scheme: "exec", Opaque: "/usr/local/bin/test.sh"},
			Status:    api.StatusHealthy,
			Message:   "foobar",
			Latency:   123 * time.Millisecond,
		},
		api.Record{
			CheckedAt: time.Now().Add(10 * time.Second),
			Target:    &url.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:    api.StatusFailure,
			Message:   "hoge",
			Latency:   123 * time.Microsecond,
		},
		api.Record{
			CheckedAt: time.Now(),
			Target:    &url.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:    api.StatusHealthy,
			Message:   "hoge",
			Latency:   123 * time.Microsecond,
		},
	}

	for _, r := range records {
		s1.Report(r)
	}

	time.Sleep(100 * time.Millisecond) // wait for write

	s2, err := store.New(f.Name())
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	s2.Console = io.Discard
	defer s2.Close()

	if err = s2.Restore(); err != nil {
		t.Fatalf("failed to restore store: %s", err)
	}

	hs1 := s1.ProbeHistory()
	hs2 := s2.ProbeHistory()

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
}

func TestStore_AddTarget(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	if len(s.ProbeHistory()) != 0 {
		t.Fatalf("found unexpected probe history")
	}

	s.Report(api.Record{
		Target:  &url.URL{Scheme: "dummy", Fragment: "add-target-1"},
		Message: "already exists history",
		Status:  api.StatusHealthy,
	})
	if len(s.ProbeHistory()) != 1 {
		t.Fatalf("found unexpected probe history")
	}

	s.AddTarget(&url.URL{Scheme: "dummy", Fragment: "add-target-2"})
	s.AddTarget(&url.URL{Scheme: "dummy", Fragment: "add-target-1"})
	s.AddTarget(&url.URL{Scheme: "dummy", Fragment: "add-target-2"})

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
	s.OnIncident = []store.IncidentHandler{
		func(s *string) func(*api.Incident) {
			return func(i *api.Incident) {
				*s = i.Message
			}
		}(&lastIncident),
	}

	assertLastIncident := func(s *string) func(string) {
		return func(expect string) {
			if *s != expect {
				t.Fatalf("expected last incident is %#v but got %#v", expect, *s)
			}
		}
	}(&lastIncident)
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
	appendRecord := func(fragment, message string, status api.Status) {
		t.Helper()
		offset += 1 * time.Second

		s.Report(api.Record{
			CheckedAt: time.Now().Add(offset),
			Target:    &url.URL{Scheme: "dummy", Fragment: fragment},
			Message:   message,
			Status:    status,
		})
	}

	appendRecord("incident-test-1", "1-1", api.StatusHealthy)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory())
	assertLastIncident("")

	appendRecord("incident-test-1", "1-2", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-1")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("1-2")

	appendRecord("incident-test-1", "1-2", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-1")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("1-2")

	appendRecord("incident-test-2", "2-1", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-1", "dummy:#incident-test-2")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("2-1")

	appendRecord("incident-test-1", "1-3", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-2", "dummy:#incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy:#incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-1", api.StatusFailure)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-2", "dummy:#incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy:#incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-?", api.StatusAborted)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-2", "dummy:#incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy:#incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-1", "1-4", api.StatusHealthy)
	assertIncidents(s.CurrentIncidents(), "dummy:#incident-test-2")
	assertIncidents(s.IncidentHistory(), "dummy:#incident-test-1", "dummy:#incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-2", api.StatusHealthy)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory(), "dummy:#incident-test-1", "dummy:#incident-test-1", "dummy:#incident-test-2")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-?", api.StatusAborted)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory(), "dummy:#incident-test-1", "dummy:#incident-test-1", "dummy:#incident-test-2")
	assertLastIncident("1-3")
}

func TestStore_incident_len_limit(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	for i := 0; i < store.INCIDENT_HISTORY_LEN*2; i++ {
		s.Report(api.Record{
			Target:  &url.URL{Scheme: "dummy", Fragment: "history-limit-test"},
			Message: fmt.Sprintf("incident-%d", i),
			Status:  api.StatusFailure,
		})
	}

	if len(s.IncidentHistory()) != store.INCIDENT_HISTORY_LEN {
		t.Fatalf("unexpected incident history length: %d (expected maximum is %d)", len(s.IncidentHistory()), store.INCIDENT_HISTORY_LEN)
	}
}

func BenchmarkStore_Append(b *testing.B) {
	for _, status := range []api.Status{api.StatusHealthy, api.StatusFailure} {
		b.Run(status.String(), func(b *testing.B) {
			s := testutil.NewStore(b)
			defer s.Close()

			record := api.Record{
				CheckedAt: time.Now(),
				Target:    &url.URL{Scheme: "dummy", Fragment: "benchmark-append"},
				Status:    status,
				Message:   "hello world",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Report(record)
			}
		})
	}
}
