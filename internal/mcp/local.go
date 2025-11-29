package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MonitoringEntry represents a single monitoring schedule entry.
type MonitoringEntry struct {
	ID       string   `json:"id"`
	Schedule string   `json:"schedule"`
	Targets  []string `json:"targets"`
}

// TargetURLDescription provides detailed documentation for target URL parameters.
const TargetURLDescription = `Target URLs to probe. Ayd supports various protocols for monitoring different types of services.

## Supported Protocols

### HTTP/HTTPS - Web server monitoring
Check if a web server responds with a successful status code (2xx or 3xx).
- http://example.com - GET request to HTTP server
- https://example.com - GET request to HTTPS server
- http://example.com:8080/health - Custom port and path
- http://user:pass@example.com - Basic authentication
- https-head://example.com - HEAD request instead of GET
- http-post://example.com/api - POST request
- https-options://example.com - OPTIONS request

### Ping (ICMP) - Network reachability
Check if a host responds to ICMP echo request.
- ping:example.com - Ping using default IP version
- ping4:example.com - Force IPv4
- ping6:example.com - Force IPv6

### TCP - Port connectivity
Check if a TCP port is open and accepting connections.
- tcp://example.com:22 - Check if port 22 is open
- tcp4://example.com:80 - Force IPv4
- tcp6://example.com:443 - Force IPv6

### DNS - Domain name resolution
Check if a domain name resolves correctly.
- dns:example.com - Resolve using system DNS
- dns://8.8.8.8/example.com - Use specific DNS server
- dns4:example.com - Resolve A record only
- dns6:example.com - Resolve AAAA record only

### SSH - SSH server availability
Check if an SSH server is accessible.
- ssh://example.com - Default port 22
- ssh://user@example.com:2222 - Custom user and port
- ssh://example.com?fingerprint=SHA256:... - Verify host fingerprint
- ssh://example.com?identityfile=/path/to/key - Use specific key file

### FTP/FTPS - FTP server monitoring
Check if an FTP server is accessible and optionally verify file existence.
- ftp://example.com - Anonymous FTP connection
- ftps://example.com - FTP with explicit TLS
- ftp://user:pass@example.com - Authenticated connection
- ftp://example.com/path/to/file.txt - Check specific file exists

### SFTP - SSH File Transfer Protocol
Check if SFTP server is accessible and optionally verify file existence.
- sftp://user@example.com - Default port 22
- sftp://user@example.com/path/to/file - Check specific file
- sftp://example.com?identityfile=/path/to/key - Use specific key

### File - Local file existence
Check if a local file or directory exists.
- file:/path/to/file - Check file exists
- file:/path/to/directory - Check directory exists

### Exec - Execute command
Execute a local command and check its exit status. Exit code 0 means healthy.
- exec:/path/to/script.sh - Execute local script
- exec:/usr/bin/test?arg=-f&arg=/tmp/file - Pass arguments via query
- exec+ssh://user@host/path/to/script - Execute via SSH

### Source - Load targets from file/URL
Load multiple probe targets from an external source file.
- source:./targets.txt - Load from local file
- source+http://example.com/targets - Load from HTTP URL

### Plugin - External probe plugins
Use external ayd-xxx-probe executables for custom protocols.
- myprotocol:target - Calls ayd-myprotocol-probe executable

### Dummy - Testing purposes
For testing and development, always returns specified status.
- dummy: or dummy:healthy - Returns HEALTHY
- dummy:failure - Returns FAILURE
- dummy:unknown - Returns UNKNOWN
- dummy:random - Returns random status`

// CheckTargetInput is the input for check_target tool.
type CheckTargetInput struct {
	Targets []string `json:"targets" jsonschema_description:"Target URLs to probe. Ayd supports various protocols for monitoring different types of services.\n\n## Supported Protocols\n\n### HTTP/HTTPS - Web server monitoring\nCheck if a web server responds with a successful status code (2xx or 3xx).\n- http://example.com - GET request to HTTP server\n- https://example.com - GET request to HTTPS server\n- http://example.com:8080/health - Custom port and path\n- http://user:pass@example.com - Basic authentication\n- https-head://example.com - HEAD request instead of GET\n- http-post://example.com/api - POST request\n- https-options://example.com - OPTIONS request\n\n### Ping (ICMP) - Network reachability\nCheck if a host responds to ICMP echo request.\n- ping:example.com - Ping using default IP version\n- ping4:example.com - Force IPv4\n- ping6:example.com - Force IPv6\n\n### TCP - Port connectivity\nCheck if a TCP port is open and accepting connections.\n- tcp://example.com:22 - Check if port 22 is open\n- tcp4://example.com:80 - Force IPv4\n- tcp6://example.com:443 - Force IPv6\n\n### DNS - Domain name resolution\nCheck if a domain name resolves correctly.\n- dns:example.com - Resolve using system DNS\n- dns://8.8.8.8/example.com - Use specific DNS server\n- dns4:example.com - Resolve A record only\n- dns6:example.com - Resolve AAAA record only\n\n### SSH - SSH server availability\nCheck if an SSH server is accessible.\n- ssh://example.com - Default port 22\n- ssh://user@example.com:2222 - Custom user and port\n- ssh://example.com?fingerprint=SHA256:... - Verify host fingerprint\n- ssh://example.com?identityfile=/path/to/key - Use specific key file\n\n### FTP/FTPS - FTP server monitoring\nCheck if an FTP server is accessible and optionally verify file existence.\n- ftp://example.com - Anonymous FTP connection\n- ftps://example.com - FTP with explicit TLS\n- ftp://user:pass@example.com - Authenticated connection\n- ftp://example.com/path/to/file.txt - Check specific file exists\n\n### SFTP - SSH File Transfer Protocol\nCheck if SFTP server is accessible and optionally verify file existence.\n- sftp://user@example.com - Default port 22\n- sftp://user@example.com/path/to/file - Check specific file\n- sftp://example.com?identityfile=/path/to/key - Use specific key\n\n### File - Local file existence\nCheck if a local file or directory exists.\n- file:/path/to/file - Check file exists\n- file:/path/to/directory - Check directory exists\n\n### Exec - Execute command\nExecute a local command and check its exit status. Exit code 0 means healthy.\n- exec:/path/to/script.sh - Execute local script\n- exec:/usr/bin/test?arg=-f&arg=/tmp/file - Pass arguments via query\n- exec+ssh://user@host/path/to/script - Execute via SSH\n\n### Source - Load targets from file/URL\nLoad multiple probe targets from an external source file.\n- source:./targets.txt - Load from local file\n- source+http://example.com/targets - Load from HTTP URL\n\n### Plugin - External probe plugins\nUse external ayd-xxx-probe executables for custom protocols.\n- myprotocol:target - Calls ayd-myprotocol-probe executable\n\n### Dummy - Testing purposes\nFor testing and development, always returns specified status.\n- dummy: or dummy:healthy - Returns HEALTHY\n- dummy:failure - Returns FAILURE\n- dummy:unknown - Returns UNKNOWN\n- dummy:random - Returns random status"`
}

// CheckTargetOutput is the output of check_target tool.
type CheckTargetOutput struct {
	Results []map[string]any `json:"results" jsonschema:"Results of probing each target."`
}

// StartMonitoringInput is the input for start_monitoring tool.
type StartMonitoringInput struct {
	Schedule string   `json:"schedule" jsonschema_description:"Schedule for monitoring. Supports interval duration or cron-style expressions.\n\n## Interval Format\nSimple duration format for regular intervals:\n- 30s - Every 30 seconds\n- 5m - Every 5 minutes\n- 1h - Every 1 hour\n- 1h30m - Every 1 hour and 30 minutes\n\n## Cron Format\nStandard cron expression with 5 or 6 fields (minute, hour, day, month, weekday, [second]):\n- 0 * * * ? - Every hour at minute 0\n- 0 0 * * ? - Daily at midnight\n- 0 9 * * 1-5 - Weekdays at 9:00 AM\n- */15 * * * ? - Every 15 minutes\n- 0 0 1 * ? - First day of each month at midnight\n\n## Predefined Schedules\n- @hourly - Every hour (equivalent to '0 * * * ?')\n- @daily - Every day at midnight (equivalent to '0 0 * * ?')\n- @weekly - Every week on Sunday at midnight\n- @monthly - First day of each month at midnight\n- @yearly - January 1st at midnight\n\nNote: Interval schedules (e.g., '5m') will execute immediately when started, then repeat at the specified interval. Cron schedules will wait until the next scheduled time."`
	Targets  []string `json:"targets" jsonschema_description:"Target URLs to monitor. Ayd supports various protocols for monitoring different types of services.\n\n## Supported Protocols\n\n### HTTP/HTTPS - Web server monitoring\nCheck if a web server responds with a successful status code (2xx or 3xx).\n- http://example.com - GET request to HTTP server\n- https://example.com - GET request to HTTPS server\n- http://example.com:8080/health - Custom port and path\n- http://user:pass@example.com - Basic authentication\n- https-head://example.com - HEAD request instead of GET\n- http-post://example.com/api - POST request\n- https-options://example.com - OPTIONS request\n\n### Ping (ICMP) - Network reachability\nCheck if a host responds to ICMP echo request.\n- ping:example.com - Ping using default IP version\n- ping4:example.com - Force IPv4\n- ping6:example.com - Force IPv6\n\n### TCP - Port connectivity\nCheck if a TCP port is open and accepting connections.\n- tcp://example.com:22 - Check if port 22 is open\n- tcp4://example.com:80 - Force IPv4\n- tcp6://example.com:443 - Force IPv6\n\n### DNS - Domain name resolution\nCheck if a domain name resolves correctly.\n- dns:example.com - Resolve using system DNS\n- dns://8.8.8.8/example.com - Use specific DNS server\n- dns4:example.com - Resolve A record only\n- dns6:example.com - Resolve AAAA record only\n\n### SSH - SSH server availability\nCheck if an SSH server is accessible.\n- ssh://example.com - Default port 22\n- ssh://user@example.com:2222 - Custom user and port\n- ssh://example.com?fingerprint=SHA256:... - Verify host fingerprint\n- ssh://example.com?identityfile=/path/to/key - Use specific key file\n\n### FTP/FTPS - FTP server monitoring\nCheck if an FTP server is accessible and optionally verify file existence.\n- ftp://example.com - Anonymous FTP connection\n- ftps://example.com - FTP with explicit TLS\n- ftp://user:pass@example.com - Authenticated connection\n- ftp://example.com/path/to/file.txt - Check specific file exists\n\n### SFTP - SSH File Transfer Protocol\nCheck if SFTP server is accessible and optionally verify file existence.\n- sftp://user@example.com - Default port 22\n- sftp://user@example.com/path/to/file - Check specific file\n- sftp://example.com?identityfile=/path/to/key - Use specific key\n\n### File - Local file existence\nCheck if a local file or directory exists.\n- file:/path/to/file - Check file exists\n- file:/path/to/directory - Check directory exists\n\n### Exec - Execute command\nExecute a local command and check its exit status. Exit code 0 means healthy.\n- exec:/path/to/script.sh - Execute local script\n- exec:/usr/bin/test?arg=-f&arg=/tmp/file - Pass arguments via query\n- exec+ssh://user@host/path/to/script - Execute via SSH\n\n### Source - Load targets from file/URL\nLoad multiple probe targets from an external source file.\n- source:./targets.txt - Load from local file\n- source+http://example.com/targets - Load from HTTP URL\n\n### Plugin - External probe plugins\nUse external ayd-xxx-probe executables for custom protocols.\n- myprotocol:target - Calls ayd-myprotocol-probe executable\n\n### Dummy - Testing purposes\nFor testing and development, always returns specified status.\n- dummy: or dummy:healthy - Returns HEALTHY\n- dummy:failure - Returns FAILURE\n- dummy:unknown - Returns UNKNOWN\n- dummy:random - Returns random status"`
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

// singleRecordReporter captures a single Record from a probe.
type singleRecordReporter struct {
	record *api.Record
}

func (r *singleRecordReporter) Report(source *api.URL, rec api.Record) {
	if r.record == nil {
		r.record = &rec
	}
}

func (r *singleRecordReporter) DeactivateTarget(source *api.URL, targets ...*api.URL) {}

// probeTarget probes a single target and returns the result.
func probeTarget(ctx context.Context, targetURL string) api.Record {
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

// CheckTarget probes the given targets and returns the results.
func CheckTarget(ctx context.Context, input CheckTargetInput) (CheckTargetOutput, error) {
	if len(input.Targets) == 0 {
		return CheckTargetOutput{}, fmt.Errorf("at least one target URL is required")
	}

	results := make([]map[string]any, 0, len(input.Targets))

	var wg sync.WaitGroup
	resultChan := make(chan map[string]any, len(input.Targets))

	for _, target := range input.Targets {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			rec := probeTarget(ctx, t)
			resultChan <- RecordToMap(rec)
		}(target)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
	}

	return CheckTargetOutput{Results: results}, nil
}

// StartMonitoringFunc starts monitoring with the given schedule and targets.
func StartMonitoringFunc(scheduler *Scheduler, input StartMonitoringInput) (StartMonitoringOutput, error) {
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
func ListMonitoringFunc(scheduler *Scheduler, input ListMonitoringInput) (ListMonitoringOutput, error) {
	entries := scheduler.ListMonitoring(input.Keywords)
	return ListMonitoringOutput{Entries: entries}, nil
}

// StopMonitoringFunc stops the monitoring entries with the given IDs.
func StopMonitoringFunc(scheduler *Scheduler, input StopMonitoringInput) (StopMonitoringOutput, error) {
	if len(input.IDs) == 0 {
		return StopMonitoringOutput{}, fmt.Errorf("at least one ID is required")
	}

	stopped, errors := scheduler.StopMonitoring(input.IDs)
	return StopMonitoringOutput{Stopped: stopped, Errors: errors}, nil
}

// AddLocalTools adds the local-only tools to the MCP server.
// These tools are: check_target, start_monitoring, list_monitoring, stop_monitoring.
func AddLocalTools(server *mcp.Server, scheduler *Scheduler) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_target",
		Title:       "Check target",
		Description: "Check the status of targets once. This performs a one-shot probe without starting continuous monitoring.",
		Annotations: &mcp.ToolAnnotations{
			IdempotentHint: true,
			ReadOnlyHint:   false,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CheckTargetInput) (*mcp.CallToolResult, CheckTargetOutput, error) {
		output, err := CheckTarget(ctx, input)
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
