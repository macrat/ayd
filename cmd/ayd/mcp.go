package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	mcputil "github.com/macrat/ayd/internal/mcp"
	"github.com/macrat/ayd/internal/meta"
	"github.com/macrat/ayd/internal/schedule"
	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robfig/cron/v3"
	"github.com/spf13/pflag"
)

// MCPCommand represents the MCP subcommand.
type MCPCommand struct {
	OutStream io.Writer
	ErrStream io.Writer
}

var defaultMCPCommand = &MCPCommand{
	OutStream: os.Stdout,
	ErrStream: os.Stderr,
}

const MCPHelp = `Ayd mcp -- Start local MCP server for monitoring control

Usage: ayd mcp [OPTIONS...]

Options:
  -f, --log-file  Path to log file. (default "ayd_%Y%m%d.log")
  -n, --name      Instance name.
  -h, --help      Show this help message and exit.
`

func (cmd *MCPCommand) Run(args []string) int {
	flags := pflag.NewFlagSet("ayd mcp", pflag.ContinueOnError)

	logPath := flags.StringP("log-file", "f", "ayd_%Y%m%d.log", "Path to log file")
	instanceName := flags.StringP("name", "n", "", "Instance name")
	help := flags.BoolP("help", "h", false, "Show this message and exit")

	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(cmd.ErrStream, err)
		fmt.Fprintf(cmd.ErrStream, "\nPlease see `%s mcp -h` for more information.\n", args[0])
		return 2
	}

	if *help {
		io.WriteString(cmd.OutStream, MCPHelp)
		return 0
	}

	if *logPath == "-" {
		*logPath = ""
	}

	s, err := store.New(*instanceName, *logPath, io.Discard)
	if err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: failed to open log file: %s\n", err)
		return 1
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localStore := &LocalMCPStore{store: s}
	prober := &LocalMCPProber{store: s}
	scheduler := NewLocalMCPScheduler(ctx, s)
	defer scheduler.Stop()

	server := cmd.createMCPServer(*instanceName, localStore, prober, scheduler)

	// Use stdio transport for MCP
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: MCP server error: %s\n", err)
		return 1
	}

	return 0
}

func (cmd *MCPCommand) createMCPServer(instanceName string, s mcputil.Store, prober mcputil.Prober, scheduler mcputil.Scheduler) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "ayd",
		Version: meta.Version,
		Title:   "Ayd Local MCP",
	}

	opts := &mcp.ServerOptions{
		Instructions: "Ayd Local MCP server. This server provides monitoring control capabilities including checking targets, starting/stopping monitoring, and querying logs.",
	}

	if instanceName != "" {
		impl.Title = impl.Title + " (" + instanceName + ")"
		opts.Instructions = opts.Instructions + " This Ayd instance's name is \"" + instanceName + "\"."
	}

	server := mcp.NewServer(impl, opts)

	// Add read-only tools (query_status, query_incidents, query_logs)
	mcputil.AddReadOnlyTools(server, s, nil)

	// Add local-only tools (check_target, start_monitoring, list_monitoring, stop_monitoring)
	mcputil.AddLocalTools(server, prober, scheduler)

	return server
}

// LocalMCPStore implements mcputil.Store interface for local MCP.
type LocalMCPStore struct {
	store *store.Store
}

func (s *LocalMCPStore) Name() string {
	return s.store.Name()
}

func (s *LocalMCPStore) ProbeHistory() []api.ProbeHistory {
	return s.store.ProbeHistory()
}

func (s *LocalMCPStore) CurrentIncidents() []*api.Incident {
	return s.store.CurrentIncidents()
}

func (s *LocalMCPStore) IncidentHistory() []*api.Incident {
	return s.store.IncidentHistory()
}

func (s *LocalMCPStore) ReportInternalError(scope, message string) {
	s.store.ReportInternalError(scope, message)
}

func (s *LocalMCPStore) OpenLog(since, until time.Time) (api.LogScanner, error) {
	return s.store.OpenLog(since, until)
}

// LocalMCPProber implements mcputil.Prober interface for local MCP.
type LocalMCPProber struct {
	store *store.Store
}

func (p *LocalMCPProber) Probe(ctx context.Context, targetURL string) api.Record {
	prober, err := scheme.NewProber(targetURL)
	if err != nil {
		target, _ := api.ParseURL(targetURL)
		return api.Record{
			Time:    time.Now(),
			Status:  api.StatusUnknown,
			Target:  target,
			Message: fmt.Sprintf("failed to create prober: %s", err),
		}
	}

	reporter := &singleRecordReporter{}
	prober.Probe(ctx, reporter)

	if reporter.record != nil {
		return *reporter.record
	}

	return api.Record{
		Time:    time.Now(),
		Status:  api.StatusUnknown,
		Target:  prober.Target(),
		Message: "no result",
	}
}

// singleRecordReporter captures a single Record from a probe.
type singleRecordReporter struct {
	record *api.Record
}

func (r *singleRecordReporter) Report(source *api.URL, rec api.Record) {
	if r.record == nil {
		r.record = &rec
	}
}

func (r *singleRecordReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {
	// No-op for single record reporter
}

// LocalMCPScheduler implements mcputil.Scheduler interface for local MCP.
type LocalMCPScheduler struct {
	ctx     context.Context
	cron    *cron.Cron
	store   *store.Store
	entries map[string]schedulerEntry
	mu      sync.RWMutex
}

type schedulerEntry struct {
	id       cron.EntryID
	schedule string
	targets  []string
}

func NewLocalMCPScheduler(ctx context.Context, s *store.Store) *LocalMCPScheduler {
	scheduler := &LocalMCPScheduler{
		ctx:     ctx,
		cron:    cron.New(),
		store:   s,
		entries: make(map[string]schedulerEntry),
	}
	scheduler.cron.Start()
	return scheduler
}

func (s *LocalMCPScheduler) Stop() {
	s.cron.Stop()
}

func (s *LocalMCPScheduler) StartMonitoring(scheduleSpec string, targets []string) (string, error) {
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
			prober.Probe(s.ctx, s.store)
		}
	}))

	// Kick start if needed
	if sched.NeedKickWhenStart() {
		go func() {
			for _, prober := range probers {
				prober.Probe(s.ctx, s.store)
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

func (s *LocalMCPScheduler) StopMonitoring(ids []string) ([]string, []string) {
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

func (s *LocalMCPScheduler) ListMonitoring(keywords []string) []mcputil.MonitoringEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []mcputil.MonitoringEntry

	for id, entry := range s.entries {
		if matchKeywords(entry.targets, keywords) {
			entries = append(entries, mcputil.MonitoringEntry{
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
