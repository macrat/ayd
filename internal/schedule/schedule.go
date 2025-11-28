package schedule

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// CurrentTime returns current time.
// This variable is for testing purpose.
var CurrentTime = time.Now

var (
	DefaultSchedule = Schedule(IntervalSchedule{5 * time.Minute})
)

type Schedule interface {
	cron.Schedule
	fmt.Stringer

	NeedKickWhenStart() bool
}

func Parse(spec string) (Schedule, error) {
	if s, err := ParseAfter(spec); err == nil {
		return s, nil
	}

	if s, err := ParseInterval(spec); err == nil {
		return s, nil
	}

	return ParseCron(spec)
}

type IntervalSchedule struct {
	Interval time.Duration
}

func ParseInterval(spec string) (IntervalSchedule, error) {
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

func ParseCron(spec string) (CronSchedule, error) {
	switch spec {
	case "@yearly", "@annually":
		spec = "0 0 1 1 ?"
	case "@monthly":
		spec = "0 0 1 * ?"
	case "@weekly":
		spec = "0 0 * * 0"
	case "@daily":
		spec = "0 0 * * ?"
	case "@hourly":
		spec = "0 * * * ?"
	default:
		delimiter := regexp.MustCompile("[ \t]+")

		ss := delimiter.Split(strings.TrimSpace(spec), -1)
		if len(ss) == 4 {
			ss = append(ss, "?")
		}
		spec = strings.Join(ss, " ")
	}

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

type AfterSchedule struct {
	Delay time.Duration
	At    time.Time
}

func ParseAfter(spec string) (Schedule, error) {
	if spec == "@reboot" {
		return RebootSchedule{}, nil
	}

	if !strings.HasPrefix(spec, "@after ") {
		return nil, fmt.Errorf("invalid schedule spec: %q", spec)
	}

	delay, err := time.ParseDuration(strings.TrimSpace(spec[len("@after "):]))
	if err != nil {
		return nil, err
	}

	if delay < 0 {
		return nil, fmt.Errorf("invalid schedule spec: %q", spec)
	}
	if delay == 0 {
		return RebootSchedule{}, nil
	}

	return AfterSchedule{
		Delay: delay,
		At:    CurrentTime().Add(delay),
	}, nil
}

func (s AfterSchedule) Next(t time.Time) time.Time {
	if t.After(s.At) {
		return time.UnixMicro(math.MaxInt64)
	}
	return s.At
}

func (s AfterSchedule) String() string {
	return "@after " + s.Delay.String()
}

func (s AfterSchedule) NeedKickWhenStart() bool {
	return false
}

type RebootSchedule struct{}

func (s RebootSchedule) Next(t time.Time) time.Time {
	return time.UnixMicro(math.MaxInt64)
}

func (s RebootSchedule) String() string {
	return "@reboot"
}

func (s RebootSchedule) NeedKickWhenStart() bool {
	return true
}
