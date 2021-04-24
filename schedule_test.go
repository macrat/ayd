package main_test

import (
	"testing"

	"github.com/macrat/ayd"
)

func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output string
		Error  string
	}{
		{"4values", "1 2 3 4", "1 2 3 4 ?", ""},
		{"5values", "1 2 3 4 5", "1 2 3 4 5", ""},
		{"spaces", "1  2 \t3 4", "1 2 3 4 ?", ""},
		{"3values", "1 2 3", "", "expected 4 to 5 fields, found 3: [1 2 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			schedule, err := main.ParseCronSchedule(tt.Input)
			if err != nil && err.Error() != tt.Error {
				t.Fatalf("unexpected error: expected %#v but got %#v", tt.Error, err.Error())
			}
			if err == nil && tt.Error != "" {
				t.Fatalf("expected error %#v but got nil", tt.Error)
			}

			if schedule.String() != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, schedule.String())
			}
		})
	}
}
