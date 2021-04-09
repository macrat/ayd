package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

var (
	DEFAULT_SCHEDULE = SimpleSchedule(cron.Every(5 * time.Minute))
)

type Schedule interface {
	cron.Schedule
	fmt.Stringer
}

func ParseSchedule(spec string) (Schedule, error) {
	if s, err := ParseSimpleSchedule(spec); err == nil {
		return s, nil
	}

	return ParseCronSchedule(spec)
}

type SimpleSchedule cron.ConstantDelaySchedule

func ParseSimpleSchedule(spec string) (SimpleSchedule, error) {
	if d, err := time.ParseDuration(spec); err != nil {
		return SimpleSchedule{}, err
	} else {
		return SimpleSchedule(cron.Every(d)), nil
	}
}

func (s SimpleSchedule) Next(t time.Time) time.Time {
	return cron.ConstantDelaySchedule(s).Next(t)
}

func (s SimpleSchedule) String() string {
	return s.Delay.String()
}

type CronSchedule struct {
	spec     string
	schedule cron.Schedule
}

func ParseCronSchedule(spec string) (CronSchedule, error) {
	if s, err := cron.ParseStandard(spec); err != nil {
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
