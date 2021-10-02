package main

import (
	"context"
	"fmt"
	"time"

	"github.com/macrat/ayd/internal/probe"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/robfig/cron/v3"
)

type Task struct {
	Schedule Schedule
	Probe    probe.Probe
}

func (t Task) MakeJob(ctx context.Context, s *store.Store) cron.Job {
	return cron.FuncJob(func() {
		defer func() {
			if err := recover(); err != nil {
				s.Report(api.Record{
					CheckedAt: time.Now(),
					Target:    t.Probe.Target(),
					Status:    api.StatusUnknown,
					Message:   fmt.Sprintf("panic: %s", err),
				})
			}
		}()

		t.Probe.Check(ctx, s)
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

		p, err := probe.New(a)
		if err != nil {
			switch err {
			case probe.ErrUnsupportedScheme:
				err = fmt.Errorf("%s: This scheme is not supported. Please check the plugin is installed if need.", a)
			case probe.ErrMissingScheme:
				err = fmt.Errorf("%s: Not valid as schedule or target URL. Please specify scheme if this is target. (e.g. ping:example.local or http://example.com)", a)
			case probe.ErrInvalidURL:
				err = fmt.Errorf("%s: Not valid as schedule or target URL.", a)
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
