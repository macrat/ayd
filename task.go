package main

import (
	"fmt"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/robfig/cron"
)

type Task struct {
	Schedule Schedule
	Probe    probe.Probe
}

func (t Task) MakeJob(s *store.Store) cron.Job {
	return cron.FuncJob(func() {
		defer func() {
			if err := recover(); err != nil {
				s.Append(store.Record{
					CheckedAt: time.Now(),
					Target:    t.Probe.Target(),
					Status:    store.STATUS_UNKNOWN,
					Message:   fmt.Sprintf("panic: %s", err),
				})
			}
		}()
		s.Append(t.Probe.Check()...)
	})
}

func (t Task) SameAs(another Task) bool {
	return t.Schedule.String() == another.Schedule.String() && t.Probe.Target().String() == another.Probe.Target().String()
}

func (t Task) In(list []Task) bool {
	for _, x := range list {
		if t.SameAs(x) {
			return true
		}
	}
	return false
}

func ParseArgs(args []string) ([]Task, []error) {
	var tasks []Task
	var errors []error

	schedule := DEFAULT_SCHEDULE

	for _, a := range args {
		if s, err := ParseSchedule(a); err == nil {
			schedule = s
			continue
		}

		p, err := probe.Get(a)
		if err != nil {
			switch err {
			case probe.ErrUnsupportedScheme:
				err = fmt.Errorf("%s: This scheme is not supported.", a)
			case probe.ErrMissingScheme:
				err = fmt.Errorf("%s: Not valid as schedule or target URI. Please specify scheme if this is target. (e.g. ping:example.local or http://example.com)", a)
			case probe.ErrInvalidURI:
				err = fmt.Errorf("%s: Not valid as schedule or target URI.", a)
			default:
				err = fmt.Errorf("%s: %s", a, err)
			}
			errors = append(errors, err)
			continue
		}

		tasks = append(tasks, Task{
			Schedule: schedule,
			Probe:    p,
		})
	}

	var result []Task
	for _, t := range tasks {
		if t.In(result) {
			continue
		}
		result = append(result, t)
	}

	return result, errors
}
