package endpoint

import (
	"net/http"
	"time"

	"github.com/macrat/ayd/internal/mcp"
	"github.com/macrat/ayd/internal/query"
	api "github.com/macrat/ayd/lib-ayd"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

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

// MCPServer creates an MCP server for the given store.
func MCPServer(s Store) *mcpsdk.Server {
	return mcp.NewRemoteServer(s.Name(), s, logFilter)
}

// MCPHandler creates an HTTP handler for MCP requests.
func MCPHandler(s Store) http.Handler {
	server := MCPServer(s)

	handler := mcpsdk.NewStreamableHTTPHandler(func(req *http.Request) *mcpsdk.Server {
		return server
	}, &mcpsdk.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return handler
}
