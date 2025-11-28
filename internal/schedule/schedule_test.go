package schedule_test

import (
	"math"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/schedule"
)

func TestParseCron(t *testing.T) {
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
			s, err := schedule.ParseCron(tt.Input)
			if err != nil && err.Error() != tt.Error {
				t.Fatalf("unexpected error: expected %#v but got %#v", tt.Error, err.Error())
			}
			if err == nil && tt.Error != "" {
				t.Fatalf("expected error %#v but got nil", tt.Error)
			}

			if s.String() != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, s.String())
			}
		})
	}
}

func TestParseInterval(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output string
		Error  bool
	}{
		{"valid", "5m", "5m0s", false},
		{"hour", "1h", "1h0m0s", false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s, err := schedule.ParseInterval(tt.Input)
			if (err != nil) != tt.Error {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && s.String() != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, s.String())
			}
		})
	}
}

func TestParseAfter(t *testing.T) {
	type TimePair struct {
		Input time.Time
		Next  time.Time
	}

	never := time.UnixMicro(math.MaxInt64)

	// Set up test time
	testTime := time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC)
	schedule.CurrentTime = func() time.Time { return testTime }
	defer func() { schedule.CurrentTime = time.Now }()

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
			s, err := schedule.ParseAfter(tt.Input)
			if err != nil {
				t.Fatalf("failed to parse schedule: %s", err)
			}

			if tt.String != s.String() {
				t.Errorf("unexpected string: expected %#v but got %#v", tt.String, s.String())
			}

			for _, tp := range tt.Times {
				n := s.Next(tp.Input)
				if !n.Equal(tp.Next) {
					t.Errorf("unexpected next schedule for %s: expected %s but got %s", tp.Input, tp.Next, n)
				}
			}

			if s.NeedKickWhenStart() != tt.KickWhenStart {
				t.Errorf("unexpected NeedKickWhenStart value: expected %v but got %v", tt.KickWhenStart, s.NeedKickWhenStart())
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output string
		Error  bool
	}{
		{"interval", "5m", "5m0s", false},
		{"cron", "0 0 * * ?", "0 0 * * ?", false},
		{"daily", "@daily", "0 0 * * ?", false},
		{"reboot", "@reboot", "@reboot", false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			s, err := schedule.Parse(tt.Input)
			if (err != nil) != tt.Error {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && s.String() != tt.Output {
				t.Errorf("expected %#v but got %#v", tt.Output, s.String())
			}
		})
	}
}

func TestIntervalSchedule_NeedKickWhenStart(t *testing.T) {
	s, _ := schedule.ParseInterval("5m")
	if !s.NeedKickWhenStart() {
		t.Error("IntervalSchedule should need kick when start")
	}
}

func TestCronSchedule_NeedKickWhenStart(t *testing.T) {
	s, _ := schedule.ParseCron("0 0 * * ?")
	if s.NeedKickWhenStart() {
		t.Error("CronSchedule should not need kick when start")
	}
}

func TestDefaultSchedule(t *testing.T) {
	if schedule.DefaultSchedule.String() != "5m0s" {
		t.Errorf("unexpected default schedule: %s", schedule.DefaultSchedule.String())
	}
}
