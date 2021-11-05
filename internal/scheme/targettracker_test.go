package scheme_test

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestFixedSourceReporter(t *testing.T) {
	sink := &testutil.DummyReporter{}

	r := scheme.FixedSourceReporter{
		Source:    &url.URL{Scheme: "dummy", Fragment: "fixed"},
		Upstreams: []scheme.Reporter{sink},
	}

	r.Report(
		&url.URL{Scheme: "dummy", Fragment: "source1"},
		api.Record{
			Target: &url.URL{Scheme: "dummy", Fragment: "record"},
		},
	)

	r.Report(
		&url.URL{Scheme: "dummy", Fragment: "source2"},
		api.Record{
			Target: &url.URL{Scheme: "dummy", Fragment: "record"},
		},
	)

	r.Report(
		&url.URL{Scheme: "dummy", Fragment: "source3"},
		api.Record{
			Target: &url.URL{Scheme: "dummy", Fragment: "record2"},
		},
	)

	sink.Lock()
	defer sink.Unlock()

	if len(sink.Sources) != 3 {
		t.Fatalf("unexpected number of sources: %v", sink.Sources)
	}
	if len(sink.Records) != 3 {
		t.Fatalf("unexpected number of reports: %v", sink.Records)
	}

	for i, s := range sink.Sources {
		if s.String() != "dummy:#fixed" {
			t.Errorf("expected all source is dummy:#fixed but sources[%d] was %s", i, s)
		}
	}

	if sink.Records[0].Target.String() != "dummy:#record" {
		t.Errorf("unexpected 1st record's target: %s", sink.Records[0])
	}
	if sink.Records[1].Target.String() != "dummy:#record" {
		t.Errorf("unexpected 2nd record's target: %s", sink.Records[1])
	}
	if sink.Records[2].Target.String() != "dummy:#record2" {
		t.Errorf("unexpected 3rd record's target: %s", sink.Records[2])
	}
}

func TestTargetTracker(t *testing.T) {
	tracker := &scheme.TargetTracker{}

	sink := &testutil.DummyReporter{}
	r := tracker.PrepareReporter(&url.URL{Scheme: "dummy", Fragment: "tracker"}, sink)
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record1"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record1"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record2"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record3"}})
	if len(sink.Records) != 4 {
		t.Errorf("unexpected number of reports: %v", sink.Records)
	}

	inactives := []string{}
	for _, x := range tracker.Inactives() {
		inactives = append(inactives, x.String())
	}
	if diff := cmp.Diff([]string{}, inactives); diff != "" {
		t.Fatalf("unexpected inactive targets\n%s", diff)
	}

	sink = &testutil.DummyReporter{}
	r = tracker.PrepareReporter(&url.URL{Scheme: "dummy", Fragment: "tracker"}, sink)
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record2"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record1"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record3"}})
	if len(sink.Records) != 3 {
		t.Errorf("unexpected number of reports: %v", sink.Records)
	}

	inactives = []string{}
	for _, x := range tracker.Inactives() {
		inactives = append(inactives, x.String())
	}
	if diff := cmp.Diff([]string{}, inactives); diff != "" {
		t.Fatalf("unexpected inactive targets\n%s", diff)
	}

	sink = &testutil.DummyReporter{}
	r = tracker.PrepareReporter(&url.URL{Scheme: "dummy", Fragment: "tracker"}, sink)
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record2"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record2"}})
	if len(sink.Records) != 2 {
		t.Errorf("unexpected number of reports: %v", sink.Records)
	}

	inactives = []string{}
	for _, x := range tracker.Inactives() {
		inactives = append(inactives, x.String())
	}
	if diff := cmp.Diff([]string{"dummy:#record1", "dummy:#record3"}, inactives); diff != "" {
		t.Fatalf("unexpected inactive targets\n%s", diff)
	}

	sink = &testutil.DummyReporter{}
	r = tracker.PrepareReporter(&url.URL{Scheme: "dummy", Fragment: "tracker"}, sink)
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record2"}})
	r.Report(nil, api.Record{Target: &url.URL{Scheme: "dummy", Fragment: "record1"}})
	if len(sink.Records) != 2 {
		t.Errorf("unexpected number of reports: %v", sink.Records)
	}

	inactives = []string{}
	for _, x := range tracker.Inactives() {
		inactives = append(inactives, x.String())
	}
	if diff := cmp.Diff([]string{}, inactives); diff != "" {
		t.Fatalf("unexpected inactive targets\n%s", diff)
	}

	sink = &testutil.DummyReporter{}
	tracker.PrepareReporter(&url.URL{Scheme: "dummy", Fragment: "tracker"}, sink)
	if len(sink.Records) != 0 {
		t.Errorf("unexpected number of reports: %v", sink.Records)
	}

	inactives = []string{}
	for _, x := range tracker.Inactives() {
		inactives = append(inactives, x.String())
	}
	if diff := cmp.Diff([]string{"dummy:#record2", "dummy:#record1"}, inactives); diff != "" {
		t.Fatalf("unexpected inactive targets\n%s", diff)
	}
}
