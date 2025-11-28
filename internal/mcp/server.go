package mcp

import (
	"github.com/macrat/ayd/internal/meta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates a new MCP server with the given configuration.
// If includeLocalTools is true, adds local-only tools (check_target, start_monitoring, etc.).
func NewServer(instanceName string, store Store, filter LogFilterFunc, includeLocalTools bool, prober Prober, scheduler Scheduler) *mcp.Server {
	title := "Ayd"
	instructions := "Ayd is a simple alive monitoring tool. The logs and status can be large, so it is recommended to extract necessary information using search and jq queries instead of fetching all data at once."

	if includeLocalTools {
		title = "Ayd Local MCP"
		instructions = "Ayd Local MCP server. This server provides monitoring control capabilities including checking targets, starting/stopping monitoring, and querying logs."
	}

	if instanceName != "" {
		title = title + " (" + instanceName + ")"
		instructions = instructions + " This Ayd instance's name is \"" + instanceName + "\"."
	}

	impl := &mcp.Implementation{
		Name:    "ayd",
		Version: meta.Version,
		Title:   title,
	}

	opts := &mcp.ServerOptions{
		Instructions: instructions,
	}

	server := mcp.NewServer(impl, opts)

	// Add read-only tools (query_status, query_incidents, query_logs)
	AddReadOnlyTools(server, store, filter)

	// Add local-only tools if requested
	if includeLocalTools && prober != nil && scheduler != nil {
		AddLocalTools(server, prober, scheduler)
	}

	return server
}

// NewRemoteServer creates an MCP server for remote access (read-only tools only).
func NewRemoteServer(instanceName string, store Store, filter LogFilterFunc) *mcp.Server {
	return NewServer(instanceName, store, filter, false, nil, nil)
}

// NewLocalServer creates an MCP server for local access (includes local-only tools).
func NewLocalServer(instanceName string, store Store, filter LogFilterFunc, prober Prober, scheduler Scheduler) *mcp.Server {
	return NewServer(instanceName, store, filter, true, prober, scheduler)
}
