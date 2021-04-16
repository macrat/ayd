package store_test

import (
	"fmt"
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

	if len(s1.ProbeHistory) != len(s2.ProbeHistory) {
		t.Fatalf("unexpected history length: %d", len(s2.ProbeHistory))
	}

	for key, ph1 := range s1.ProbeHistory {
		ph2, ok := s2.ProbeHistory[key]
		if !ok {
			t.Errorf("restored store has no %s", key)
			continue
		}

		if len(ph1.Results) != len(ph2.Results) {
			t.Errorf("%s: unmatch restored records number: %d != %d", key, len(ph1.Results), len(ph2.Results))
			continue
		}

		for i := range ph1.Results {
			if ph1.Results[i].Equals(*ph2.Results[i]) {
				t.Errorf("%s %d: unexpected record", key, i)
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

	if len(s.ProbeHistory) != 0 {
		t.Fatalf("found unexpected probe history")
	}

	s.Append(store.Record{
		Target:  &url.URL{Scheme: "dummy", Opaque: "add-target-1"},
		Message: "already exists history",
		Status:  store.STATUS_HEALTHY,
	})
	if len(s.ProbeHistory) != 1 {
		t.Fatalf("found unexpected probe history")
	}

	s.AddTarget(&url.URL{Scheme: "dummy", Opaque: "add-target-2"})
	s.AddTarget(&url.URL{Scheme: "dummy", Opaque: "add-target-1"})
	s.AddTarget(&url.URL{Scheme: "dummy", Opaque: "add-target-2"})

	if len(s.ProbeHistory) != 2 {
		t.Fatalf("unexpected length probe history: %d", len(s.ProbeHistory))
	}

	hs := s.ProbeHistory.AsSortedArray()

	if hs[0].Target.String() != "dummy:add-target-1" {
		t.Errorf("unexpected 1st target: %s", hs[0].Target)
	}
	if len(hs[0].Results) != 1 || hs[0].Results[0].Message != "already exists history" {
		t.Errorf("1st target's record may override: %#v", hs[0].Results)
	}

	if hs[1].Target.String() != "dummy:add-target-2" {
		t.Errorf("unexpected 2nd target: %s", hs[1].Target)
	}
	if len(hs[1].Results) != 0 {
		t.Errorf("2nd target has unexpected record: %#v", hs[1].Results)
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
	s.OnIncident = func(s *string) func(*store.Incident) []store.Record {
		return func(i *store.Incident) []store.Record {
			*s = i.Message
			return nil
		}
	}(&lastIncident)

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
	appendRecord := func(opaque, message string, status store.Status) {
		t.Helper()

		s.Append(store.Record{
			Target:  &url.URL{Scheme: "dummy", Opaque: opaque},
			Message: message,
			Status:  status,
		})
	}

	appendRecord("incident-test-1", "1-1", store.STATUS_HEALTHY)
	assertIncidents(s.CurrentIncidents)
	assertIncidents(s.IncidentHistory)
	assertLastIncident("")

	appendRecord("incident-test-1", "1-2", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents, "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory)
	assertLastIncident("1-2")

	appendRecord("incident-test-1", "1-2", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents, "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory)
	assertLastIncident("1-2")

	appendRecord("incident-test-2", "2-1", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents, "dummy:incident-test-1", "dummy:incident-test-2")
	assertIncidents(s.IncidentHistory)
	assertLastIncident("2-1")

	appendRecord("incident-test-1", "1-3", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents, "dummy:incident-test-2", "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory, "dummy:incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-1", store.STATUS_FAILURE)
	assertIncidents(s.CurrentIncidents, "dummy:incident-test-2", "dummy:incident-test-1")
	assertIncidents(s.IncidentHistory, "dummy:incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-1", "1-4", store.STATUS_HEALTHY)
	assertIncidents(s.CurrentIncidents, "dummy:incident-test-2")
	assertIncidents(s.IncidentHistory, "dummy:incident-test-1", "dummy:incident-test-1")
	assertLastIncident("1-3")

	appendRecord("incident-test-2", "2-2", store.STATUS_HEALTHY)
	assertIncidents(s.CurrentIncidents)
	assertIncidents(s.IncidentHistory, "dummy:incident-test-1", "dummy:incident-test-1", "dummy:incident-test-2")
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

	if len(s.IncidentHistory) != store.INCIDENT_HISTORY_LEN {
		t.Fatalf("unexpected incident history length: %d (expected maximum is %d)", len(s.IncidentHistory), store.INCIDENT_HISTORY_LEN)
	}
}
