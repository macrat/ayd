package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	DEFAULT_SCHEDULE = Schedule(IntervalSchedule{5 * time.Minute})
)

type Schedule interface {
	cron.Schedule
	fmt.Stringer

	NeedKickWhenStart() bool
}

func ParseSchedule(spec string) (Schedule, error) {
	if s, err := ParseIntervalSchedule(spec); err == nil {
		return s, nil
	}

	return ParseCronSchedule(spec)
}

type IntervalSchedule struct {
	Interval time.Duration
}

func ParseIntervalSchedule(spec string) (IntervalSchedule, error) {
	if d, err := time.ParseDuration(spec); err != nil {
		return IntervalSchedule{}, err
	} else {
		return IntervalSchedule{d}, nil
	}
}

func (s IntervalSchedule) Next(t time.Time) time.Time {
	return t.Add(s.Interval)
}

func (s IntervalSchedule) String() string {
	return s.Interval.String()
}

func (s IntervalSchedule) NeedKickWhenStart() bool {
	return true
}

type CronSchedule struct {
	spec     string
	schedule cron.Schedule
}

func ParseCronSchedule(spec string) (CronSchedule, error) {
	delimiter := regexp.MustCompile("[ \t]+")

	ss := delimiter.Split(strings.TrimSpace(spec), -1)
	if len(ss) == 4 {
		ss = append(ss, "?")
	}
	spec = strings.Join(ss, " ")

	if s, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional).Parse(spec); err != nil {
		return CronSchedule{}, err
	} else {
		return CronSchedule{
			spec:     spec,
			schedule: s,
		}, nil
	}
}

func (s CronSchedule) Next(t time.Time) time.Time {
	return s.schedule.Next(t)
}

func (s CronSchedule) String() string {
	return s.spec
}

func (s CronSchedule) NeedKickWhenStart() bool {
	return false
}
