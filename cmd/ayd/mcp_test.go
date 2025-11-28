package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMCPCommand_Run_help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := &MCPCommand{
		OutStream: &stdout,
		ErrStream: &stderr,
	}

	code := cmd.Run([]string{"ayd", "mcp", "-h"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(stdout.String(), "Ayd mcp") {
		t.Errorf("expected help text in stdout, got: %s", stdout.String())
	}
}

func TestMCPCommand_Run_invalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := &MCPCommand{
		OutStream: &stdout,
		ErrStream: &stderr,
	}

	code := cmd.Run([]string{"ayd", "mcp", "--invalid-flag"})
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}

	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Errorf("expected error message in stderr, got: %s", stderr.String())
	}
}

func TestMCPCommand_Run_disableLog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := &MCPCommand{
		OutStream: &stdout,
		ErrStream: &stderr,
	}

	// Test that -f - with -h works correctly
	code := cmd.Run([]string{"ayd", "mcp", "-f", "-", "-h"})
	if code != 0 {
		t.Errorf("expected exit code 0 with -h, got %d", code)
	}
}

func TestMCPHelp(t *testing.T) {
	// Verify help text contains expected information
	if !strings.Contains(MCPHelp, "ayd mcp") {
		t.Error("help text should contain 'ayd mcp'")
	}
	if !strings.Contains(MCPHelp, "--log-file") {
		t.Error("help text should contain '--log-file'")
	}
	if !strings.Contains(MCPHelp, "--name") {
		t.Error("help text should contain '--name'")
	}
	if !strings.Contains(MCPHelp, "--help") {
		t.Error("help text should contain '--help'")
	}
}
