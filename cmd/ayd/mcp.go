package main

import (
	"context"
	"fmt"
	"io"
	"os"

	mcputil "github.com/macrat/ayd/internal/mcp"
	"github.com/macrat/ayd/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

	scheduler := mcputil.NewScheduler(ctx, s)
	defer scheduler.Stop()

	server := mcputil.NewLocalServer(*instanceName, s, scheduler)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: MCP server error: %s\n", err)
		return 1
	}

	return 0
}
