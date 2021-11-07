package store

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
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
