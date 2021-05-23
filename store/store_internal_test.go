package store

import (
	"net/url"
	"reflect"
	"sort"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestByLatestStatus(t *testing.T) {
	xs := []*ProbeHistory{
		{
			&url.URL{Scheme: "a", Opaque: "1"},
			[]api.Record{},
		},
		{
			&url.URL{Scheme: "a", Opaque: "2"},
			[]api.Record{{Status: api.StatusHealthy}},
		},
		{
			&url.URL{Scheme: "a", Opaque: "3"},
			[]api.Record{{Status: api.StatusHealthy}},
		},
		{
			&url.URL{Scheme: "a", Opaque: "4"},
			[]api.Record{{Status: api.StatusFailure}},
		},
		{
			&url.URL{Scheme: "b", Opaque: "1"},
			[]api.Record{{Status: api.StatusUnknown}},
		},
		{
			&url.URL{Scheme: "b", Opaque: "2"},
			[]api.Record{{Status: api.StatusAborted}},
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
