package mcp

import (
	"github.com/macrat/ayd/internal/meta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newServer creates a new MCP server with the given configuration.
func newServer(instanceName string, store Store, filter LogFilterFunc, includeLocalTools bool, scheduler Scheduler) *mcp.Server {
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
	if includeLocalTools && scheduler != nil {
		AddLocalTools(server, scheduler)
	}

	return server
}

// NewRemoteServer creates an MCP server for remote access (read-only tools only).
func NewRemoteServer(instanceName string, store Store, filter LogFilterFunc) *mcp.Server {
	return newServer(instanceName, store, filter, false, nil)
}

// NewLocalServer creates an MCP server for local access (includes local-only tools).
func NewLocalServer(instanceName string, store Store, filter LogFilterFunc, scheduler Scheduler) *mcp.Server {
	return newServer(instanceName, store, filter, true, scheduler)
}
