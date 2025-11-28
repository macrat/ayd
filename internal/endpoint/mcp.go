package endpoint

import (
	"net/http"
	"time"

	mcputil "github.com/macrat/ayd/internal/mcp"
	"github.com/macrat/ayd/internal/meta"
	"github.com/macrat/ayd/internal/query"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPOutput is an alias for mcp.Output for backward compatibility.
type MCPOutput = mcputil.Output

// ParseJQ parses a jq query string.
func ParseJQ(q string) (mcputil.JQQuery, error) {
	return mcputil.ParseJQ(q)
}

// MCPStatusInput is an alias for mcp.StatusInput for backward compatibility.
type MCPStatusInput = mcputil.StatusInput

// MCPIncidentsInput is an alias for mcp.IncidentsInput for backward compatibility.
type MCPIncidentsInput = mcputil.IncidentsInput

// MCPLogsInput is an alias for mcp.LogsInput for backward compatibility.
type MCPLogsInput = mcputil.LogsInput

// storeAdapter adapts endpoint.Store to mcp.Store interface.
type storeAdapter struct {
	Store
}

func (a storeAdapter) Name() string {
	return a.Store.Name()
}

func (a storeAdapter) ProbeHistory() []api.ProbeHistory {
	return a.Store.ProbeHistory()
}

func (a storeAdapter) CurrentIncidents() []*api.Incident {
	return a.Store.CurrentIncidents()
}

func (a storeAdapter) IncidentHistory() []*api.Incident {
	return a.Store.IncidentHistory()
}

func (a storeAdapter) ReportInternalError(scope, message string) {
	a.Store.ReportInternalError(scope, message)
}

func (a storeAdapter) OpenLog(since, until time.Time) (api.LogScanner, error) {
	return a.Store.OpenLog(since, until)
}

// logFilter is a filter function for query_logs tool.
func logFilter(scanner api.LogScanner, search string) (api.LogScanner, time.Time, time.Time) {
	q := query.ParseQuery(search)
	st, en := q.TimeRange()

	var since, until time.Time
	if st != nil {
		since = *st
	}
	if en != nil {
		until = *en
	}

	return FilterScanner{
		Scanner: scanner,
		Query:   q,
	}, since, until
}

func MCPServer(s Store) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "ayd",
		Version: meta.Version,
		Title:   "Ayd",
	}

	opts := &mcp.ServerOptions{
		Instructions: "Ayd is a simple alive monitoring tool. The logs and status can be large, so it is recommended to extract necessary information using search and jq queries instead of fetching all data at once.",
	}

	if s.Name() != "" {
		impl.Title = impl.Title + " (" + s.Name() + ")"
		opts.Instructions = opts.Instructions + " This Ayd instance's name is \"" + s.Name() + "\"."
	}

	server := mcp.NewServer(impl, opts)

	mcputil.AddReadOnlyTools(server, storeAdapter{s}, logFilter)

	return server
}

func MCPHandler(s Store) http.Handler {
	server := MCPServer(s)

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return handler
}
