package main_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/macrat/ayd"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/testutil"
)

type PanicProbe struct{}

func (p PanicProbe) Target() *url.URL {
	return &url.URL{Scheme: "test", Opaque: "panic"}
}

func (p PanicProbe) Check(ctx context.Context, r probe.Reporter) {
	panic("this always make panic")
}

func TestMakeJob(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	task := main.Task{Probe: PanicProbe{}}
	task.MakeJob(context.Background(), s).Run()

	history := s.ProbeHistory()[0]

	if len(history.Records) != 1 {
		t.Fatalf("unexpected length history found\nhistory.Records = %#v", history.Records)
	}

	r := history.Records[0]

	if r.Status != api.StatusUnknown {
		t.Errorf("unexpected status: %s", r.Status)
	}
	if r.Message != "panic: this always make panic" {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestParseArgs(t *testing.T) {
	type WantTask struct {
		Schedule string
		Target   string
	}

	tests := []struct {
		Args []string
		Want []WantTask
	}{
		{
			Args: []string{"ping:hoge", "2m", "http://example.com"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"2m0s", "http://example.com"},
			},
		},
		{
			Args: []string{"ping:hoge", "ping://fuga", "2m", "1h", "http://example.com"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"5m0s", "ping:fuga"},
				{"1h0m0s", "http://example.com"},
			},
		},
		{
			Args: []string{"ping:hoge", "2m", "ping:hoge", "5m", "ping:hoge?abc", "2m", "ping://hoge"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"2m0s", "ping:hoge"},
			},
		},
		{
			Args: []string{"*/10  * * *", "ping:hoge", "*/5 *\t* * 0", "ping:fuga"},
			Want: []WantTask{
				{"*/10 * * * ?", "ping:hoge"},
				{"*/5 * * * 0", "ping:fuga"},
			},
		},
	}

	for _, tt := range tests {
		result, errs := main.ParseArgs(tt.Args)
		if errs != nil {
			for _, err := range errs {
				t.Errorf("%#v: failed parse: %s", tt.Args, err)
			}
			continue
		}

		if len(result) != len(tt.Want) {
			t.Errorf("%#v: expected %d tasks but got %d tasks", tt.Args, len(tt.Want), len(result))
			continue
		}

		for i := range result {
			if result[i].Probe.Target().String() != tt.Want[i].Target {
				t.Errorf("%#v: unexpected target at %d: expected %#v but got %#v", tt.Args, i, tt.Want[i].Target, result[i].Probe.Target().String())
			}

			if result[i].Schedule.String() != tt.Want[i].Schedule {
				t.Errorf("%#v: unexpected interval at %d: expected %s but got %s", tt.Args, i, tt.Want[i].Schedule, result[i].Schedule)
			}
		}
	}
}
