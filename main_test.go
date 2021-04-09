package main_test

import (
	"testing"

	"github.com/macrat/ayd"
)

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
			Args: []string{"hoge", "2m", "http://example.com"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"2m0s", "http://example.com"},
			},
		},
		{
			Args: []string{"hoge", "fuga", "2m", "1h", "http://example.com"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"5m0s", "ping:fuga"},
				{"1h0m0s", "http://example.com"},
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
