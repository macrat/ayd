package store

import (
	"testing"
	"time"
)

func Test_yearFragment_Match(t *testing.T) {
	tests := []struct {
		str   string
		since time.Time
		until time.Time
		want  bool
	}{
		{"2022", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), true},
		{"2021", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"2023", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"22", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), true},
		{"21", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"23", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"-123", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"no", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
	}

	for _, tt := range tests {
		actual := yearFragment{len(tt.str) == 2}.Match(tt.str, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: want=%v actual=%v", tt.str, tt.want, actual)
		}
	}
}

func Test_monthFragment_Match(t *testing.T) {
	tests := []struct {
		str   string
		since time.Time
		until time.Time
		want  bool
	}{
		{"01", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), true},
		{"12", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"06", time.Date(2021, 7, 0, 0, 0, 0, 0, time.UTC), time.Date(2022, 9, 0, 0, 0, 0, 0, time.UTC), true},
		{"07", time.Date(2021, 1, 0, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 0, 0, 0, 0, 0, time.UTC), true},
		{"00", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"20", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
		{"no", time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), time.Date(2022, 1, 2, 15, 4, 5, 6, time.UTC), false},
	}

	for _, tt := range tests {
		actual := monthFragment{}.Match(tt.str, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: want=%v actual=%v", tt.str, tt.want, actual)
		}
	}
}

func Test_dayFragment_Match(t *testing.T) {
	tests := []struct {
		str   string
		since time.Time
		until time.Time
		want  bool
	}{
		{"01", time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC), false},
		{"02", time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC), true},
		{"03", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 2, 4, 0, 0, 0, 0, time.UTC), true},
		{"04", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"10", time.Date(2022, 1, 11, 0, 0, 0, 0, time.UTC), time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC), true},
		{"11", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC), true},
		{"30", time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC), false},
		{"31", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"32", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"-1", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"no", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		actual := dayFragment{}.Match(tt.str, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: want=%v actual=%v", tt.str, tt.want, actual)
		}
	}
}

func Test_hourFragment_Match(t *testing.T) {
	tests := []struct {
		str   string
		since time.Time
		until time.Time
		want  bool
	}{
		{"01", time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC), true},
		{"02", time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC), true},
		{"03", time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC), false},
		{"04", time.Date(2022, 1, 1, 5, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 18, 0, 0, 0, time.UTC), false},
		{"05", time.Date(2022, 1, 1, 18, 0, 0, 0, time.UTC), time.Date(2022, 1, 2, 18, 0, 0, 0, time.UTC), true},
		{"06", time.Date(2022, 1, 1, 4, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC), false},
		{"07", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 2, 1, 0, 0, 0, time.UTC), true},
		{"08", time.Date(2022, 1, 1, 23, 0, 0, 0, time.UTC), time.Date(2022, 1, 2, 9, 0, 0, 0, time.UTC), true},
		{"22", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 2, 1, 0, 0, 0, time.UTC), true},
		{"23", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 20, 0, 0, 0, time.UTC), false},
		{"24", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"-1", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"no", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		actual := hourFragment{}.Match(tt.str, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: want=%v actual=%v", tt.str, tt.want, actual)
		}
	}
}

func Test_minuteFragment_Match(t *testing.T) {
	tests := []struct {
		str   string
		since time.Time
		until time.Time
		want  bool
	}{
		{"01", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 59, 0, 0, time.UTC), true},
		{"02", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC), true},
		{"03", time.Date(2022, 1, 1, 0, 59, 0, 0, time.UTC), time.Date(2022, 1, 1, 1, 10, 0, 0, time.UTC), true},
		{"04", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 3, 0, 0, time.UTC), false},
		{"05", time.Date(2022, 1, 1, 0, 6, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 7, 0, 0, time.UTC), false},
		{"06", time.Date(2022, 1, 1, 0, 6, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 7, 0, 0, time.UTC), true},
		{"07", time.Date(2022, 1, 1, 0, 5, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 7, 0, 0, time.UTC), true},
		{"08", time.Date(2022, 1, 1, 0, 5, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 7, 0, 0, time.UTC), false},
		{"09", time.Date(2022, 1, 1, 0, 10, 0, 0, time.UTC), time.Date(2022, 1, 1, 1, 7, 0, 0, time.UTC), false},
		{"10", time.Date(2022, 1, 1, 0, 15, 0, 0, time.UTC), time.Date(2022, 1, 1, 1, 11, 0, 0, time.UTC), true},
		{"11", time.Date(2022, 1, 1, 0, 10, 0, 0, time.UTC), time.Date(2022, 1, 1, 1, 10, 0, 0, time.UTC), true},
		{"-1", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"no", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		actual := minuteFragment{}.Match(tt.str, tt.since, tt.until)
		if actual != tt.want {
			t.Errorf("%s: want=%v actual=%v", tt.str, tt.want, actual)
		}
	}
}
