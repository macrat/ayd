package mcp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/itchyny/gojq"
)

// jqParseURL is a custom jq function to parse URLs.
func jqParseURL(x any, _ []any) any {
	str, ok := x.(string)
	if !ok {
		return fmt.Errorf("parse_url/0: expected a string but got %T (%v)", x, x)
	}
	u, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("parse_url/0: failed to parse URL: %v", err)
	}

	username := ""
	if u.User != nil {
		username = u.User.Username()
	}

	queries := map[string][]any{}
	for key, vals := range u.Query() {
		queries[key] = make([]any, len(vals))
		for i, v := range vals {
			queries[key][i] = v
		}
	}

	if u.Opaque != "" && u.Host == "" {
		switch u.Scheme {
		case "ping", "ping4", "ping6":
			u.Host = u.Opaque
			u.Opaque = ""
		case "dns", "dns4", "dns6", "file", "exec", "mailto", "source":
			u.Path = u.Opaque
			u.Opaque = ""
		}
	}

	return map[string]any{
		"scheme":   u.Scheme,
		"username": username,
		"hostname": u.Hostname(),
		"port":     u.Port(),
		"path":     u.Path,
		"queries":  queries,
		"fragment": u.Fragment,
		"opaque":   u.Opaque,
	}
}

// JQQuery represents a compiled jq query.
type JQQuery struct {
	Code *gojq.Code
}

// ParseJQ parses a jq query string.
func ParseJQ(query string) (JQQuery, error) {
	if query == "" {
		query = "."
	}

	q, err := gojq.Parse(query)
	if err != nil {
		return JQQuery{}, err
	}

	c, err := gojq.Compile(
		q,
		gojq.WithFunction("parse_url", 0, 0, jqParseURL),
	)
	if err != nil {
		return JQQuery{}, err
	}

	return JQQuery{Code: c}, nil
}

// Output represents the result of an MCP tool call.
type Output struct {
	Result any `json:"result" jsonschema:"The result of the query."`
}

// Run executes the jq query on the input and returns the result.
func (q JQQuery) Run(ctx context.Context, input any) (Output, error) {
	var outputs []any

	iter := q.Code.RunWithContext(ctx, input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if halt, ok := v.(*gojq.HaltError); ok {
			if halt.ExitCode() == 0 {
				break
			}
			v := map[string]any{
				"status":    "halt_error",
				"exit_code": halt.ExitCode(),
				"value":     halt.Value(),
			}
			outputs = append(outputs, v)
			break
		} else if err, ok := v.(error); ok {
			return Output{}, err
		}
		outputs = append(outputs, v)
	}

	if len(outputs) == 1 {
		return Output{
			Result: outputs[0],
		}, nil
	} else {
		return Output{
			Result: outputs,
		}, nil
	}
}
