package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestProbeHistory_sources(t *testing.T) {
	ph := &probeHistory{}

	add := func(opaque string) {
		ph.addSource(&api.URL{Scheme: "dummy", Opaque: opaque})
	}
	remove := func(opaque string) {
		ph.removeSource(&api.URL{Scheme: "dummy", Opaque: opaque})
	}
	assert := func(sources ...string) {
		t.Helper()

		for i, x := range sources {
			sources[i] = "dummy:" + x
		}

		diff := cmp.Diff(sources, ph.sources)
		if diff != "" {
			t.Fatalf("unexpected sources\n%s", diff)
		}
	}

	assert()

	add("hello")
	add("hello")
	add("hello")
	assert("hello")

	add("world")
	assert("hello", "world")

	add("foo")
	add("bar")
	assert("hello", "world", "foo", "bar")

	remove("world")
	assert("hello", "foo", "bar")

	ph.setInactive()
	assert()
}

func TestProbeHistoryMap(t *testing.T) {
	m := make(probeHistoryMap)

	for i := 1; i <= 100; i++ {
		m.Append(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    time.Now().Add(time.Duration(i) * time.Second),
			Target:  &api.URL{Scheme: "dummy", Fragment: "append-test"},
			Message: fmt.Sprint(i),
		})
	}

	if hs, ok := m["dummy:#append-test"]; !ok {
		t.Errorf("failed to get history\n%#v", m)
	} else if len(hs.Records) != PROBE_HISTORY_LEN {
		t.Errorf("unexpected number of records: %d", len(hs.Records))
	} else if hs.Records[len(hs.Records)-1].Message != "100" {
		t.Errorf("unexpected message of latest record: %#v", hs.Records[len(hs.Records)-1])
	}

	for i := 1; i <= 10; i++ {
		m.Append(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    time.Now().Add(time.Duration(i) * time.Second),
			Target:  &api.URL{Scheme: "dummy", Fragment: "append-test-another"},
			Message: fmt.Sprint(i),
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
		m.Append(&api.URL{Scheme: "dummy"}, api.Record{
			Time:    time.Now().Add(time.Duration(-i) * time.Second),
			Target:  &api.URL{Scheme: "dummy", Fragment: "append-test-reverse"},
			Message: fmt.Sprint(i),
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
	m.Append(&api.URL{Scheme: "dummy"}, api.Record{
		Time:    timestamp,
		Target:  &api.URL{Scheme: "dummy", Fragment: "append-test-same-time"},
		Message: "first",
	})
	m.Append(&api.URL{Scheme: "dummy"}, api.Record{
		Time:    timestamp,
		Target:  &api.URL{Scheme: "dummy", Fragment: "append-test-same-time"},
		Message: "second",
	})

	if hs, ok := m["dummy:#append-test-same-time"]; !ok {
		t.Errorf("failed to get history\n%#v", m)
	} else if len(hs.Records) != 2 {
		t.Errorf("unexpected number of records: %d", len(hs.Records))
	} else if hs.Records[len(hs.Records)-1].Message != "second" {
		t.Errorf("unexpected message of latest record: %#v", hs.Records[len(hs.Records)-1])
	}
}

func TestProbeHistoryMap_isActive(t *testing.T) {
	m := make(probeHistoryMap)
	target := &api.URL{Scheme: "dummy", Fragment: "is-active-test"}

	if m.isActive(target) {
		t.Fatalf("unexpected active on empty map")
	}

	m.Append(target, api.Record{Time: time.Now(), Target: target})

	if !m.isActive(target) {
		t.Fatalf("expected active after append")
	}
}

func BenchmarkProbeHistory_sources(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			ph := &probeHistory{}

			xs := make([]*api.URL, n)
			for i := range xs {
				xs[i] = &api.URL{Scheme: "dummy", Opaque: fmt.Sprint(i)}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ph.removeSource(xs[i%n])
				ph.addSource(xs[i%n])
			}
		})
	}
}
