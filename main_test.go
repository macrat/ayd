package main_test

import (
	"testing"
	"time"

	"github.com/macrat/ayd"
)

func TestParseArgs(t *testing.T) {
	type WantTask struct {
		Interval time.Duration
		Target   string
	}

	tests := []struct {
		Args []string
		Want []WantTask
	}{
		{
			Args: []string{"hoge", "2m", "http://example.com"},
			Want: []WantTask{
				{5 * time.Minute, "ping:hoge"},
				{2 * time.Minute, "http://example.com"},
			},
		},
		{
			Args: []string{"hoge", "fuga", "2m", "1h", "http://example.com"},
			Want: []WantTask{
				{5 * time.Minute, "ping:hoge"},
				{5 * time.Minute, "ping:fuga"},
				{1 * time.Hour, "http://example.com"},
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

			if result[i].Interval != tt.Want[i].Interval {
				t.Errorf("%#v: unexpected interval at %d: expected %s but got %s", tt.Args, i, tt.Want[i].Interval, result[i].Interval)
			}
		}
	}
}
