package main_test

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/store"
)

type PanicProbe struct{}

func (p PanicProbe) Target() *url.URL {
	return &url.URL{Scheme: "test", Opaque: "panic"}
}

func (p PanicProbe) Check(ctx context.Context) []store.Record {
	panic("this always make panic")
}

func TestMakeJob(t *testing.T) {
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

	task := main.Task{Probe: PanicProbe{}}
	task.MakeJob(context.Background(), s).Run()

	history, ok := s.ProbeHistory["test:panic"]
	if !ok {
		t.Fatalf("history was not found:\ns.ProbeHistory = %#v", s.ProbeHistory)
	}

	if len(history.Records) != 1 {
		t.Fatalf("unexpected length history found\nhistory.Records = %#v", history.Records)
	}

	r := history.Records[0]

	if r.Status != store.STATUS_UNKNOWN {
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
