package endpoint

import (
	"net/http"

	"github.com/macrat/ayd/internal/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer creates an MCP server for the given store.
func MCPServer(s Store) *mcpsdk.Server {
	return mcp.NewRemoteServer(s.Name(), s)
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
