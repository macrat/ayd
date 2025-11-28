package main

import (
	"github.com/macrat/ayd/internal/schedule"
)

var (
	DEFAULT_SCHEDULE = schedule.DefaultSchedule
)

// Schedule is an alias for schedule.Schedule
type Schedule = schedule.Schedule

// IntervalSchedule is an alias for schedule.IntervalSchedule
type IntervalSchedule = schedule.IntervalSchedule

// CronSchedule is an alias for schedule.CronSchedule
type CronSchedule = schedule.CronSchedule

// AfterSchedule is an alias for schedule.AfterSchedule
type AfterSchedule = schedule.AfterSchedule

// RebootSchedule is an alias for schedule.RebootSchedule
type RebootSchedule = schedule.RebootSchedule

// ParseSchedule parses schedule specification.
func ParseSchedule(spec string) (Schedule, error) {
	return schedule.Parse(spec)
}

// ParseIntervalSchedule parses interval schedule like "5m" or "1h".
func ParseIntervalSchedule(spec string) (IntervalSchedule, error) {
	return schedule.ParseInterval(spec)
}

// ParseCronSchedule parses cron schedule like "0 0 * * ?" or "@daily".
func ParseCronSchedule(spec string) (CronSchedule, error) {
	return schedule.ParseCron(spec)
}

// ParseAfterSchedule parses after schedule like "@after 30m" or "@reboot".
func ParseAfterSchedule(spec string) (Schedule, error) {
	return schedule.ParseAfter(spec)
}
