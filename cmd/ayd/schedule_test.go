package main_test

import (
	"math"
	"testing"
	"time"

	"github.com/macrat/ayd/cmd/ayd"
)

func TestParseIntervalSchedule(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output time.Duration
		Error  string
	}{
		{"valid", "10s", 10 * time.Second, ""},
		{"invalid", "abc", 0, "time: invalid duration \"abc\""},
		{"zero", "0h", 0, "interval duration: \"0h\""},
		{"negative", "-5m", 0, "interval duration: \"-5m\""},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			schedule, err := main.ParseIntervalSchedule(tt.Input)
			if err != nil && err.Error() != tt.Error {
				t.Fatalf("unexpected error: expected %#v but got %#v", tt.Error, err.Error())
			}

			if schedule.Interval != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, schedule.Interval)
			}
		})
	}
}

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
		{"@yearly", "@yearly", "0 0 1 1 ?", ""},
		{"@annually", "@annually", "0 0 1 1 ?", ""},
		{"@monthly", "@monthly", "0 0 1 * ?", ""},
		{"@weekly", "@weekly", "0 0 * * 0", ""},
		{"@daily", "@daily", "0 0 * * ?", ""},
		{"@hourly", "@hourly", "0 * * * ?", ""},
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

func TestAfterSchedule(t *testing.T) {
	type TimePair struct {
		Input time.Time
		Next  time.Time
	}

	never := time.UnixMicro(math.MaxInt64)

	tests := []struct {
		Input         string
		String        string
		Times         []TimePair
		KickWhenStart bool
	}{
		{
			Input:  "@reboot",
			String: "@reboot",
			Times: []TimePair{
				{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), never},
			},
			KickWhenStart: true,
		},
		{
			Input:  "@after 0m",
			String: "@reboot",
			Times: []TimePair{
				{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), never},
			},
			KickWhenStart: true,
		},
		{
			Input:  "@after   5m",
			String: "@after 5m0s",
			Times: []TimePair{
				{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2001, 2, 3, 16, 10, 6, 0, time.UTC)},
				{time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2001, 2, 3, 16, 10, 6, 0, time.UTC)},
				{time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC), never},
			},
			KickWhenStart: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			schedule, err := main.ParseAfterSchedule(tt.Input)
			if err != nil {
				t.Fatalf("failed to parse schedule: %s", err)
			}

			if tt.String != schedule.String() {
				t.Errorf("unexpected string: expected %#v but got %#v", tt.String, schedule.String())
			}

			for _, tp := range tt.Times {
				n := schedule.Next(tp.Input)
				if !n.Equal(tp.Next) {
					t.Errorf("unexpected next schedule for %s: expected %s but got %s", tp.Input, tp.Next, n)
				}
			}

			if schedule.NeedKickWhenStart() != tt.KickWhenStart {
				t.Errorf("unexpected NeedKickWhenStart value: expected %v but got %v", tt.KickWhenStart, schedule.NeedKickWhenStart())
			}
		})
	}
}
