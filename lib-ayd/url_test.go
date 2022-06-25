package ayd_test

import (
	"testing"

	"github.com/macrat/ayd/lib-ayd"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
		Host   string
		Path   string
		Opaque string
	}{
		{"dummy:", "dummy:", "", "", ""},
		{"dummy:healthy", "dummy:healthy", "", "", "healthy"},
		{"exec:///path/to/file", "exec:///path/to/file", "", "/path/to/file", ""},
		{"exec:/path/to/file", "exec:/path/to/file", "", "", "/path/to/file"},
		{"source+exec:/path/to/file", "source+exec:/path/to/file", "", "", "/path/to/file"},
		{"https://example.com/path/to", "https://example.com/path/to", "example.com", "/path/to", ""},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			u, err := ayd.ParseURL(tt.Input)
			if err != nil {
				t.Fatalf("failed to parse: %s", err)
			}

			if s := u.String(); s != tt.Output {
				t.Errorf("unexpected String() output\nexpected: %s\n but got: %s", tt.Output, s)
			}

			if u.Host != tt.Host {
				t.Errorf("unexpected Host\nexpected: %s\n but got: %s", tt.Host, u.Host)
			}

			if u.Path != tt.Path {
				t.Errorf("unexpected Path\nexpected: %s\n but got: %s", tt.Path, u.Path)
			}

			if u.Opaque != tt.Opaque {
				t.Errorf("unexpected Opaque\nexpected: %s\n but got: %s", tt.Opaque, u.Opaque)
			}
		})
	}
}

func TestURL_String(t *testing.T) {
	tests := []struct {
		Input  ayd.URL
		Output string
	}{
		{
			ayd.URL{Scheme: "dummy", Fragment: "Aaあ亜"},
			"dummy:#Aaあ亜",
		},
		{
			ayd.URL{Scheme: "https", Host: "テスト.com", RawQuery: "あ=亜"},
			"https://%E3%83%86%E3%82%B9%E3%83%88.com?あ=亜",
		},
		{
			ayd.URL{Scheme: "dummy", Fragment: "ああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああ"},
			"dummy:#ああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああああ",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input.String(), func(t *testing.T) {
			result := tt.Input.String()
			if tt.Output != result {
				t.Errorf("expected output is %s but got %s", tt.Output, result)
			}
		})
	}
}

func TestURL_marshalAndUnmarshal(t *testing.T) {
	tests := []struct{
		Input  string
		Output string
	} {
		{"https://example.com", "https://example.com"},
		{"file:/path/to/file.txt#hello world", "file:/path/to/file.txt#hello%20world"},
		{"a:b/c?d=e#f", "a:b/c?d=e#f"},
		{"a:/b/c?d=e#f", "a:/b/c?d=e#f"},
		{"a://b/c?d=e#f", "a://b/c?d=e#f"},
		{"a://b:c@d", "a://b:xxxxx@d"},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			var u ayd.URL

			if err := u.UnmarshalText([]byte(tt.Input)); err != nil {
				t.Fatalf("failed to unmarshal URL: %s", err)
			}

			bs, err := u.MarshalText()
			if err != nil {
				t.Fatalf("failed to marshal URL: %s", err)
			}

			if string(bs) != tt.Output {
				t.Errorf("expected output is %q but got %q", tt.Output, string(bs))
			}
		})
	}
}
