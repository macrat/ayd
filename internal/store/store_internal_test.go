package store

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestProbeHistory_sources(t *testing.T) {
	ph := &ProbeHistory{}

	add := func(opaque string) {
		ph.addSource(&url.URL{Scheme: "dummy", Opaque: opaque})
	}
	remove := func(opaque string) {
		ph.removeSource(&url.URL{Scheme: "dummy", Opaque: opaque})
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

func BenchmarkProbeHistory_sources(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			ph := &ProbeHistory{}

			xs := make([]*url.URL, n)
			for i := range xs {
				xs[i] = &url.URL{Scheme: "dummy", Opaque: fmt.Sprint(i)}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ph.removeSource(xs[i%n])
				ph.addSource(xs[i%n])
			}
		})
	}
}

func TestByLatestStatus(t *testing.T) {
	xs := []*ProbeHistory{
		{
			&url.URL{Scheme: "a", Opaque: "1"},
			[]api.Record{},
			[]string{"a:1"},
		},
		{
			&url.URL{Scheme: "a", Opaque: "2"},
			[]api.Record{{Status: api.StatusHealthy}},
			[]string{"a:2"},
		},
		{
			&url.URL{Scheme: "a", Opaque: "3"},
			[]api.Record{{Status: api.StatusHealthy}},
			[]string{"a:3"},
		},
		{
			&url.URL{Scheme: "a", Opaque: "4"},
			[]api.Record{{Status: api.StatusFailure}},
			[]string{"a:3"},
		},
		{
			&url.URL{Scheme: "b", Opaque: "1"},
			[]api.Record{{Status: api.StatusUnknown}},
			[]string{"b:1"},
		},
		{
			&url.URL{Scheme: "b", Opaque: "2"},
			[]api.Record{{Status: api.StatusAborted}},
			[]string{"b:2"},
		},
	}

	sort.Sort(byLatestStatus(xs))

	var ss []string
	for _, x := range xs {
		ss = append(ss, x.Target.String())
	}

	want := []string{"a:4", "b:1", "a:1", "a:2", "a:3", "b:2"}
	if !reflect.DeepEqual(ss, want) {
		t.Errorf("unexpected sorted result:\nexpected: %v\n but got: %v", want, ss)
	}
}
