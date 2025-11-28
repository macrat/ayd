package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MonitoringEntry represents a single monitoring schedule entry.
type MonitoringEntry struct {
	ID       string   `json:"id"`
	Schedule string   `json:"schedule"`
	Targets  []string `json:"targets"`
}

// Scheduler is an interface for managing monitoring schedules.
type Scheduler interface {
	// StartMonitoring starts monitoring with the given schedule and targets.
	// It returns the ID of the new monitoring entry.
	StartMonitoring(schedule string, targets []string) (string, error)

	// StopMonitoring stops the monitoring with the given IDs.
	// It returns the list of successfully stopped IDs and any errors.
	StopMonitoring(ids []string) ([]string, []string)

	// ListMonitoring returns all monitoring entries.
	// If keywords are provided, only entries matching ALL keywords are returned.
	ListMonitoring(keywords []string) []MonitoringEntry
}

// Prober is an interface for checking targets.
type Prober interface {
	// Probe executes a probe on the given target URL and returns the result.
	Probe(ctx context.Context, targetURL string) api.Record
}

// CheckTargetInput is the input for check_target tool.
type CheckTargetInput struct {
	Targets []string `json:"targets" jsonschema:"URLs to check. Each URL will be probed once."`
}

// CheckTargetOutput is the output of check_target tool.
type CheckTargetOutput struct {
	Results []map[string]any `json:"results" jsonschema:"Results of probing each target."`
}

// StartMonitoringInput is the input for start_monitoring tool.
type StartMonitoringInput struct {
	Schedule string   `json:"schedule" jsonschema:"Cron-style schedule or interval duration. Examples: '5m' (every 5 minutes), '0 0 * * ?' (daily at midnight), '@hourly', '@daily'."`
	Targets  []string `json:"targets" jsonschema:"URLs to monitor."`
}

// StartMonitoringOutput is the output of start_monitoring tool.
type StartMonitoringOutput struct {
	ID string `json:"id" jsonschema:"The ID of the new monitoring entry. Use this ID to stop the monitoring."`
}

// ListMonitoringInput is the input for list_monitoring tool.
type ListMonitoringInput struct {
	Keywords []string `json:"keywords,omitempty" jsonschema:"Keywords to filter monitoring entries. All keywords must match (AND condition). If omitted, all entries are returned."`
}

// ListMonitoringOutput is the output of list_monitoring tool.
type ListMonitoringOutput struct {
	Entries []MonitoringEntry `json:"entries" jsonschema:"List of monitoring entries."`
}

// StopMonitoringInput is the input for stop_monitoring tool.
type StopMonitoringInput struct {
	IDs []string `json:"ids" jsonschema:"IDs of monitoring entries to stop."`
}

// StopMonitoringOutput is the output of stop_monitoring tool.
type StopMonitoringOutput struct {
	Stopped []string `json:"stopped" jsonschema:"IDs that were successfully stopped."`
	Errors  []string `json:"errors,omitempty" jsonschema:"IDs that could not be stopped with error messages."`
}

// CheckTarget probes the given targets and returns the results.
func CheckTarget(ctx context.Context, prober Prober, input CheckTargetInput) (CheckTargetOutput, error) {
	if len(input.Targets) == 0 {
		return CheckTargetOutput{}, fmt.Errorf("at least one target URL is required")
	}

	results := make([]map[string]any, 0, len(input.Targets))

	var wg sync.WaitGroup
	resultChan := make(chan map[string]any, len(input.Targets))
	var errCount atomic.Int32

	for _, target := range input.Targets {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			rec := prober.Probe(ctx, t)
			resultChan <- RecordToMap(rec)
		}(target)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
		if result["status"] != "HEALTHY" {
			errCount.Add(1)
		}
	}

	return CheckTargetOutput{Results: results}, nil
}

// StartMonitoringFunc starts monitoring with the given schedule and targets.
func StartMonitoringFunc(scheduler Scheduler, input StartMonitoringInput) (StartMonitoringOutput, error) {
	if input.Schedule == "" {
		return StartMonitoringOutput{}, fmt.Errorf("schedule is required")
	}
	if len(input.Targets) == 0 {
		return StartMonitoringOutput{}, fmt.Errorf("at least one target URL is required")
	}

	id, err := scheduler.StartMonitoring(input.Schedule, input.Targets)
	if err != nil {
		return StartMonitoringOutput{}, err
	}

	return StartMonitoringOutput{ID: id}, nil
}

// ListMonitoringFunc returns all monitoring entries that match the keywords.
func ListMonitoringFunc(scheduler Scheduler, input ListMonitoringInput) (ListMonitoringOutput, error) {
	entries := scheduler.ListMonitoring(input.Keywords)
	return ListMonitoringOutput{Entries: entries}, nil
}

// StopMonitoringFunc stops the monitoring entries with the given IDs.
func StopMonitoringFunc(scheduler Scheduler, input StopMonitoringInput) (StopMonitoringOutput, error) {
	if len(input.IDs) == 0 {
		return StopMonitoringOutput{}, fmt.Errorf("at least one ID is required")
	}

	stopped, errors := scheduler.StopMonitoring(input.IDs)
	return StopMonitoringOutput{Stopped: stopped, Errors: errors}, nil
}

// AddLocalTools adds the local-only tools to the MCP server.
// These tools are: check_target, start_monitoring, list_monitoring, stop_monitoring.
func AddLocalTools(server *mcp.Server, prober Prober, scheduler Scheduler) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_target",
		Title:       "Check target",
		Description: "Check the status of targets once. This performs a one-shot probe without starting continuous monitoring.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   false,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CheckTargetInput) (*mcp.CallToolResult, CheckTargetOutput, error) {
		output, err := CheckTarget(ctx, prober, input)
		return nil, output, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_monitoring",
		Title:       "Start monitoring",
		Description: "Start monitoring targets with the specified schedule. Returns an ID that can be used to stop the monitoring.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: false,
			ReadOnlyHint:   false,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StartMonitoringInput) (*mcp.CallToolResult, StartMonitoringOutput, error) {
		output, err := StartMonitoringFunc(scheduler, input)
		return nil, output, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_monitoring",
		Title:       "List monitoring",
		Description: "List all active monitoring entries. Optionally filter by keywords.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   true,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListMonitoringInput) (*mcp.CallToolResult, ListMonitoringOutput, error) {
		output, err := ListMonitoringFunc(scheduler, input)
		return nil, output, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_monitoring",
		Title:       "Stop monitoring",
		Description: "Stop monitoring entries by their IDs.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   false,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StopMonitoringInput) (*mcp.CallToolResult, StopMonitoringOutput, error) {
		output, err := StopMonitoringFunc(scheduler, input)
		return nil, output, err
	})
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
