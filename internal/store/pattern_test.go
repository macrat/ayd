package store_test

import (
	"testing"
	"time"

	"github.com/macrat/ayd/internal/store"
)

func TestPattern_parseAndBuild(t *testing.T) {
	times := []time.Time{
		time.Date(2001, 2, 3, 4, 5, 6, 7, time.UTC),
		time.Date(1234, 11, 29, 20, 42, 50, 234, time.UTC),
	}
	tests := []struct {
		input string
		want  []string
	}{
		{"ayd.log", []string{"ayd.log", "ayd.log"}},
		{"ayd_%Y%m%d%H%M.log", []string{"ayd_200102030405.log", "ayd_123411292042.log"}},
		{"year=%y/month=%m/day=%d/ayd.log", []string{"year=01/month=02/day=03/ayd.log", "year=34/month=11/day=29/ayd.log"}},
		{"ayd_%ignore%%%Y.log", []string{"ayd_%ignore%2001.log", "ayd_%ignore%1234.log"}},
	}

	for _, tt := range tests {
		p := store.ParsePattern(tt.input)

		for i, want := range tt.want {
			actual := p.Build(times[i])
			if actual != want {
				t.Errorf("%s: unexpected result:\nexpected: %s\n but got: %s", tt.input, want, actual)
			}
		}
	}
}

func TestPattern_Match(t *testing.T) {
	tests := []struct {
		pattern string
		fname   string
		since   time.Time
		until   time.Time
		want    bool
	}{
		{"ayd.log", "ayd.log", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"ayd.log", "log.json", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"ayd_%Y%m%d.log", "ayd_20220101.log", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"ayd_%Y%m%d.log", "ayd_2022010.log", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"ayd_%Y-%m-%d.log", "ayd_2022-06-15.log", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"ayd_%m%d.log", "ayd_0401.log", time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC), true},
		{"%y/ayd_%H%M.log", "22/ayd_2059.log", time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 21, 0, 0, 0, time.UTC), true},
		{"%y/ayd_%H%M.log", "22/ayd_2101.log", time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 21, 0, 0, 0, time.UTC), false},
		{"%y/ayd_%H%M.log", "22/ayd_1959.log", time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 21, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		p := store.ParsePattern(tt.pattern)
		actual := p.Match(tt.fname, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: %s: want=%v actual=%v", tt.pattern, tt.fname, tt.want, actual)
		}
	}
}

func FuzzPattern_Build(f *testing.F) {
	f.Add("ayd_%y%m%d.log")
	f.Add("%Y-%m-%dT%H:%M.txt")
	f.Add("ayd/year=%Y/month=%m/day=%d/hour=%H/minute=%M/log.json")
	f.Add("ayd-log/date=%Y%m%d/time=%H%M/ayd.log")
	f.Add("/var/log/ayd/%Y/%m/%d/%H%M.log")
	f.Add("%%percent%%%")

	now := time.Now()

	f.Fuzz(func(t *testing.T, s string) {
		if len(s) == 0 {
			t.Skip()
		}

		p := store.ParsePattern(s)
		if len(p.Build(now)) == 0 {
			t.Errorf("pattern %q made an empty string", s)
		}
	})
}
