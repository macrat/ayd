package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/macrat/ayd/internal/schedule"
	"github.com/macrat/ayd/internal/scheme"
	"github.com/robfig/cron/v3"
)

// Scheduler manages monitoring schedules for local MCP.
type Scheduler struct {
	ctx      context.Context
	cron     *cron.Cron
	reporter scheme.Reporter
	entries  map[string]schedulerEntry
	mu       sync.RWMutex
}

type schedulerEntry struct {
	id       cron.EntryID
	schedule string
	targets  []string
}

// NewScheduler creates a new Scheduler.
func NewScheduler(ctx context.Context, reporter scheme.Reporter) *Scheduler {
	scheduler := &Scheduler{
		ctx:      ctx,
		cron:     cron.New(),
		reporter: reporter,
		entries:  make(map[string]schedulerEntry),
	}
	scheduler.cron.Start()
	return scheduler
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// StartMonitoring starts monitoring with the given schedule and targets.
func (s *Scheduler) StartMonitoring(scheduleSpec string, targets []string) (string, error) {
	sched, err := schedule.Parse(scheduleSpec)
	if err != nil {
		return "", fmt.Errorf("invalid schedule: %w", err)
	}

	id := uuid.New().String()

	var probers []scheme.Prober
	for _, target := range targets {
		prober, err := scheme.NewProber(target)
		if err != nil {
			return "", fmt.Errorf("invalid target %q: %w", target, err)
		}
		probers = append(probers, prober)
	}

	// Schedule using the parsed schedule
	entryID := s.cron.Schedule(sched, cron.FuncJob(func() {
		for _, prober := range probers {
			prober.Probe(s.ctx, s.reporter)
		}
	}))

	// Kick start if needed
	if sched.NeedKickWhenStart() {
		go func() {
			for _, prober := range probers {
				prober.Probe(s.ctx, s.reporter)
			}
		}()
	}

	s.mu.Lock()
	s.entries[id] = schedulerEntry{
		id:       entryID,
		schedule: scheduleSpec,
		targets:  targets,
	}
	s.mu.Unlock()

	return id, nil
}

// StopMonitoring stops the monitoring with the given IDs.
func (s *Scheduler) StopMonitoring(ids []string) ([]string, []string) {
	var stopped, errors []string

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		if entry, ok := s.entries[id]; ok {
			s.cron.Remove(entry.id)
			delete(s.entries, id)
			stopped = append(stopped, id)
		} else {
			errors = append(errors, fmt.Sprintf("%s: not found", id))
		}
	}

	return stopped, errors
}

// ListMonitoring returns all monitoring entries.
func (s *Scheduler) ListMonitoring(keywords []string) []MonitoringEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []MonitoringEntry

	for id, entry := range s.entries {
		if matchKeywords(entry.targets, keywords) {
			entries = append(entries, MonitoringEntry{
				ID:       id,
				Schedule: entry.schedule,
				Targets:  entry.targets,
			})
		}
	}

	return entries
}

// matchKeywords checks if all keywords are present in any of the targets.
func matchKeywords(targets []string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}

	for _, keyword := range keywords {
		found := false
		for _, target := range targets {
			if strings.Contains(target, keyword) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
