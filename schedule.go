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

type SimpleSchedule cron.ConstantDelaySchedule

func ParseSimpleSchedule(s string) (SimpleSchedule, error) {
	if d, err := time.ParseDuration(s); err != nil {
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
