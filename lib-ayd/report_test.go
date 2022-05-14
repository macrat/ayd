package ayd_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/macrat/ayd/lib-ayd"
)

func TestReport_TargetURLs(t *testing.T) {
	urls := []*ayd.URL{
		{Scheme: "dummy", Fragment: "hello"},
		{Scheme: "dummy", Fragment: "world"},
	}

	var want []string
	r := ayd.Report{ProbeHistory: make(map[string]ayd.ProbeHistory)}

	for _, u := range urls {
		want = append(want, u.String())
		r.ProbeHistory[u.String()] = ayd.ProbeHistory{Target: u}
	}

	var got []string
	for _, u := range r.TargetURLs() {
		got = append(got, u.String())
	}

	sort.Strings(want)
	sort.Strings(got)

	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected result\nexpected: %#v\n but got: %#v", want, got)
	}
}
