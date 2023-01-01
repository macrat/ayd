package store_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/store"
)

func TestPathPattern_parseAndBuild(t *testing.T) {
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
		p := store.ParsePathPattern(tt.input)

		for i, want := range tt.want {
			actual := p.Build(times[i])
			if actual != want {
				t.Errorf("%s: unexpected result:\nexpected: %s\n but got: %s", tt.input, want, actual)
			}
		}
	}
}

func TestPathPattern_Match(t *testing.T) {
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
		{"%Y%m%dT%H%M.log", "20210102T1504.log", time.Date(2021, 1, 2, 15, 4, 0, 0, time.UTC), time.Date(2021, 1, 2, 15, 4, 10, 0, time.UTC), true},
		{"%y/ayd_%H%M.log", "22/ayd_2059.log", time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 21, 0, 0, 0, time.UTC), true},
		{"%y/ayd_%H%M.log", "22/ayd_2101.log", time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 21, 0, 0, 0, time.UTC), false},
		{"%y/ayd_%H%M.log", "22/ayd_1959.log", time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 21, 0, 0, 0, time.UTC), false},
		{"%m_%d_%Y.log", "12_25_2001.log", time.Date(2001, 12, 1, 0, 0, 0, 0, time.UTC), time.Date(2001, 12, 30, 0, 0, 0, 0, time.UTC), true},
		{"ayd/date=%Y%m%d/time=%H%M/log.json", "ayd/year=20220203/time=1504/log.json", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 2, 3, 15, 3, 0, 0, time.UTC), false},
		{"ayd/date=%Y%m%d/time=%H%M/log.json", "ayd/year=20220203/time=1503/log.json", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 2, 3, 15, 3, 0, 0, time.UTC), false},
		{"", "", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%", "%", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%%", "%", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%%%", "%%", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%a%%", "%a%", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%Y", "2022", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%Y", "-123", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%m", "12", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), true},
		{"%m", "13", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), false},
		{"%d", "31", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), true},
		{"%d", "32", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), false},
		{"%H", "23", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), true},
		{"%H", "24", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), false},
		{"%M", "59", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), true},
		{"%M", "60", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 12, 31, 23, 59, 0, 0, time.UTC), false},
		{"%Y%m%d", "20220101", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%Y%m%d", "20220101?", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%Y%m%d", "2022010", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%Y-%Y", "2022-2022", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%Y-%Y", "2022-2023", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%Y-%y", "2022-22", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%Y-%y", "2022-23", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%m-%m", "01-01", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%m-%m", "01-02", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%d-%d", "01-01", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%d-%d", "01-02", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%H-%H", "01-01", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%H-%H", "01-02", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"%M-%M", "01-01", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"%M-%M", "01-02", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		p := store.ParsePathPattern(tt.pattern)
		actual := p.Match(tt.fname, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: %s: want=%v actual=%v", tt.pattern, tt.fname, tt.want, actual)
		}
	}
}

func TestPathPattern_ListAll(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "2021"), 0755)
	os.Mkdir(filepath.Join(dir, "2022"), 0755)
	os.WriteFile(filepath.Join(dir, "2021", "01-02.log"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "2021", "02-03.log"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "2022", "01-02.log"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "2022", "04-01.log"), []byte{}, 0644)

	p := store.ParsePathPattern(filepath.Join(dir, "%Y/%m-%d.log"))

	want := []string{
		filepath.Join(dir, "2021", "01-02.log"),
		filepath.Join(dir, "2021", "02-03.log"),
		filepath.Join(dir, "2022", "01-02.log"),
		filepath.Join(dir, "2022", "04-01.log"),
	}
	if actual := p.ListAll(); !reflect.DeepEqual(want, actual) {
		t.Errorf("unexpected files found\nwant:\n%s\nactual:\n%s", strings.Join(want, "\n"), strings.Join(actual, "\n"))
	}
}

func FuzzPathPattern_Build(f *testing.F) {
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

		p := store.ParsePathPattern(s)
		if len(p.Build(now)) == 0 {
			t.Errorf("pattern %q made an empty string", s)
		}
	})
}
