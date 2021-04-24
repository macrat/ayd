package main_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/store"
)

func TestRunOneshot(t *testing.T) {
	tests := []struct {
		Args []string
		Code int
	}{
		{[]string{"exec:echo#with-healthy", "exec:echo#::status::healthy", "exec:echo#hello"}, 0},
		{[]string{"exec:echo#with-failure", "exec:echo#::status::failure", "exec:echo#hello"}, 1},
		{[]string{"exec:echo#with-unknown", "exec:echo#::status::unknown", "exec:echo#hello"}, 2},
		{[]string{"exec:echo#single-target"}, 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.Args), func(t *testing.T) {
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

			tasks, errs := main.ParseArgs(tt.Args)
			if errs != nil {
				t.Fatalf("unexpected errors: %s", errs)
			}

			code := main.RunOneshot(s, tasks)
			if code != tt.Code {
				t.Errorf("unexpected exit code: %d", code)
			}

			count := 0
			for _, xs := range s.ProbeHistory {
				count += len(xs.Records)
			}

			if count != len(tt.Args) {
				t.Errorf("unexpected number of probe history: %d", count)
			}
		})
	}
}
