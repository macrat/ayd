package store_test

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/store"
)

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
	defer s1.Close()

	records := []store.Record{
		store.Record{
			CheckedAt: time.Now().Add(30 * time.Second),
			Target:    &url.URL{Scheme: "ping", Opaque: "restore-test"},
			Status:    store.STATUS_UNKNOWN,
			Message:   "hello world",
			Latency:   1 * time.Second,
		},
		store.Record{
			CheckedAt: time.Now().Add(20 * time.Second),
			Target:    &url.URL{Scheme: "exec", Opaque: "/usr/local/bin/test.sh"},
			Status:    store.STATUS_HEALTHY,
			Message:   "foobar",
			Latency:   123 * time.Millisecond,
		},
		store.Record{
			CheckedAt: time.Now().Add(10 * time.Second),
			Target:    &url.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:    store.STATUS_FAILURE,
			Message:   "hoge",
			Latency:   123 * time.Microsecond,
		},
		store.Record{
			CheckedAt: time.Now(),
			Target:    &url.URL{Scheme: "http", Host: "test.local", Path: "/abc/def"},
			Status:    store.STATUS_HEALTHY,
			Message:   "hoge",
			Latency:   123 * time.Microsecond,
		},
	}

	s1.Append(records...)

	s2, err := store.New(f.Name())
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
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
			if ph1.Records[j].Equals(*ph2.Records[j]) {
				t.Errorf("%d %d: unexpected record", i, j)
			}
		}
	}
}

func TestStore_AddTarget(t *testing.T) {
	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	s, err := store.New(f.Name())
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	if len(s.ProbeHistory()) != 0 {
		t.Fatalf("found unexpected probe history")
	}

	s.Append(store.Record{
		Target:  &url.URL{Scheme: "dummy", Opaque: "add-target-1"},
		Message: "already exists history",
		Status:  store.STATUS_HEALTHY,
	})
	if len(s.ProbeHistory()) != 1 {
		t.Fatalf("found unexpected probe history")
	}

	s.AddTarget(&url.URL{Scheme: "dummy", Opaque: "add-target-2"})
	s.AddTarget(&url.URL{Scheme: "dummy", Opaque: "add-target-1"})
	s.AddTarget(&url.URL{Scheme: "dummy", Opaque: "add-target-2"})

	if len(s.ProbeHistory()) != 2 {
		t.Fatalf("unexpected length probe history: %d", len(s.ProbeHistory()))
	}

	hs := s.ProbeHistory()

	if hs[0].Target.String() != "dummy:add-target-1" {
		t.Errorf("unexpected 1st target: %s", hs[0].Target)
	}
	if len(hs[0].Records) != 1 || hs[0].Records[0].Message != "already exists history" {
		t.Errorf("1st target's record may override: %#v", hs[0].Records)
	}

	if hs[1].Target.String() != "dummy:add-target-2" {
		t.Errorf("unexpected 2nd target: %s", hs[1].Target)
	}
	if len(hs[1].Records) != 0 {
		t.Errorf("2nd target has unexpected record: %#v", hs[1].Records)
	}
}

func TestStore_incident(t *testing.T) {
	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	s, err := store.New(f.Name())
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	lastIncident := ""
	s.OnIncident = []store.IncidentHandler{
		func(s *string) func(*store.Incident) []store.Record {
			return func(i *store.Incident) []store.Record {
				*s = i.Message
				return nil
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
	assertIncidents := func(incidents []*store.Incident, target ...string) {
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
	appendRecord := func(opaque, message string, status store.Status) {
		t.Helper()
		offset += 1 * time.Second

		s.Append(store.Record{
			CheckedAt: time.Now().Add(offset),
			Target:    &url.URL{Scheme: "dummy", Opaque: opaque},
			Message:   message,
			Status:    status,
		})
	}

	appendRecord("incident-test-1", "1-1", store.STATUS_HEALTHY)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory())
	assertLastIncident("")

	appendRecord("incident-test-1", "1-2", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents(), "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("1-2")

	appendRecord("incident-test-1", "1-2", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents(), "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("1-2")

	appendRecord("incident-test-2", "2-1", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents(), "dummy:incident-test-1", "dummy:incident-test-2")
	assertIncidents(s.IncidentHistory())
	assertLastIncident("2-1")

	appendRecord("incident-test-1", "1-3", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents(), "dummy:incident-test-2", "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy:incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-1", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents(), "dummy:incident-test-2", "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory(), "dummy:incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-1", "1-4", store.STATUS_HEALTHY)
	assertIncidents(s.CurrentIncidents(), "dummy:incident-test-2")
	assertIncidents(s.IncidentHistory(), "dummy:incident-test-1", "dummy:incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-2", store.STATUS_HEALTHY)
	assertIncidents(s.CurrentIncidents())
	assertIncidents(s.IncidentHistory(), "dummy:incident-test-1", "dummy:incident-test-1", "dummy:incident-test-2")
	assertLastIncident("1-3")
}

func TestStore_incident_len_limit(t *testing.T) {
	f, err := os.CreateTemp("", "ayd-test-*")
	if err != nil {
		t.Fatalf("failed to create log file: %s", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	s, err := store.New(f.Name())
	if err != nil {
		t.Fatalf("failed to create store: %s", err)
	}
	defer s.Close()

	for i := 0; i < store.INCIDENT_HISTORY_LEN*2; i++ {
		s.Append(store.Record{
			Target:  &url.URL{Scheme: "dummy", Opaque: "history-limit-test"},
			Message: fmt.Sprintf("incident-%d", i),
			Status:  store.STATUS_FAILURE,
		})
	}

	if len(s.IncidentHistory()) != store.INCIDENT_HISTORY_LEN {
		t.Fatalf("unexpected incident history length: %d (expected maximum is %d)", len(s.IncidentHistory()), store.INCIDENT_HISTORY_LEN)
	}
}

func BenchmarkStore_Append(b *testing.B) {
	for _, status := range []store.Status{store.STATUS_HEALTHY, store.STATUS_FAILURE} {
		b.Run(status.String(), func(b *testing.B) {
			f, err := os.CreateTemp("", "ayd-test-*")
			if err != nil {
				b.Fatalf("failed to create log file: %s", err)
			}
			defer os.Remove(f.Name())
			f.Close()

			s, err := store.New(f.Name())
			if err != nil {
				b.Fatalf("failed to create store: %s", err)
			}
			s.Console = io.Discard
			defer s.Close()

			record := store.Record{
				CheckedAt: time.Now(),
				Target:    &url.URL{Scheme: "dummy", Opaque: "benchmark-append"},
				Status:    status,
				Message:   "hello world",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Append(record)
			}
		})
	}
}
