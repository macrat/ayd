package mcp_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/mcp"
)

func TestParseJQ(t *testing.T) {
	tests := []struct {
		Name    string
		Query   string
		Input   any
		Output  any
		IsError bool
	}{
		{
			Name:   "empty",
			Query:  "",
			Input:  map[string]any{"foo": "bar"},
			Output: map[string]any{"foo": "bar"},
		},
		{
			Name:   "identity",
			Query:  ".",
			Input:  map[string]any{"foo": "bar"},
			Output: map[string]any{"foo": "bar"},
		},
		{
			Name:   "select",
			Query:  ".foo",
			Input:  map[string]any{"foo": "bar"},
			Output: "bar",
		},
		{
			Name:   "filter_single",
			Query:  ".[] | select(.x > 1)",
			Input:  []any{map[string]any{"x": 1}, map[string]any{"x": 2}},
			Output: map[string]any{"x": 2}, // single result is not wrapped in array
		},
		{
			Name:   "filter_multiple",
			Query:  ".[] | select(.x > 0)",
			Input:  []any{map[string]any{"x": 1}, map[string]any{"x": 2}},
			Output: []any{map[string]any{"x": 1}, map[string]any{"x": 2}}, // multiple results are wrapped in array
		},
		{
			Name:    "parse_error",
			Query:   "invalid{{",
			IsError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			q, err := mcp.ParseJQ(tt.Query)
			if tt.IsError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output, err := q.Run(context.Background(), tt.Input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.Output, output.Result); diff != "" {
				t.Errorf("unexpected output (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseURL(t *testing.T) {
	q, err := mcp.ParseJQ("parse_url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output, err := q.Run(context.Background(), "https://user@example.com:8080/path?q=v#frag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := output.Result.(map[string]any)

	if result["scheme"] != "https" {
		t.Errorf("unexpected scheme: %v", result["scheme"])
	}
	if result["username"] != "user" {
		t.Errorf("unexpected username: %v", result["username"])
	}
	if result["hostname"] != "example.com" {
		t.Errorf("unexpected hostname: %v", result["hostname"])
	}
	if result["port"] != "8080" {
		t.Errorf("unexpected port: %v", result["port"])
	}
	if result["path"] != "/path" {
		t.Errorf("unexpected path: %v", result["path"])
	}
	if result["fragment"] != "frag" {
		t.Errorf("unexpected fragment: %v", result["fragment"])
	}
}

func TestParseURL_OpaqueSchemes(t *testing.T) {
	tests := []struct {
		URL      string
		Hostname string
		Path     string
	}{
		{"ping:example.com", "example.com", ""},
		{"dns:example.com", "", "example.com"},
		{"file:/path/to/file", "", "/path/to/file"},
	}

	for _, tt := range tests {
		t.Run(tt.URL, func(t *testing.T) {
			q, err := mcp.ParseJQ("parse_url")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output, err := q.Run(context.Background(), tt.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := output.Result.(map[string]any)

			if result["hostname"] != tt.Hostname {
				t.Errorf("unexpected hostname: %v", result["hostname"])
			}
			if result["path"] != tt.Path {
				t.Errorf("unexpected path: %v", result["path"])
			}
		})
	}
}
