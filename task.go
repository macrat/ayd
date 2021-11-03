package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
	"github.com/macrat/ayd/internal/url/probe"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/robfig/cron/v3"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
)

type Task struct {
	Schedule Schedule
	Probe    probe.Probe
}

func (t Task) MakeJob(ctx context.Context, s *store.Store) cron.Job {
	return cron.FuncJob(func() {
		defer func() {
			if err := recover(); err != nil {
				s.Report(t.Probe.Target(), api.Record{
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

func ParseArgs(args []string) ([]Task, error) {
	var tasks []Task
	errors := &ayderr.ListBuilder{What: ErrInvalidArgument}

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
				errors.Pushf("%s: This scheme is not supported. Please check if the plugin is installed if need.", a)
			case probe.ErrMissingScheme:
				errors.Pushf("%s: Not valid as schedule or target URL. Please specify scheme if this is target. (e.g. ping:%s or http://%s)", a, a, a)
			case probe.ErrInvalidURL:
				errors.Pushf("%s: Not valid as schedule or target URL.", a)
			default:
				errors.Pushf("%s: %w", a, err)
			}
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

	return result, errors.Build()
}
