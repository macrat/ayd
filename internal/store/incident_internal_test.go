package store

import (
	"testing"
)

func TestIsSameIncidentMessage(t *testing.T) {
	tests := []struct {
		A, B string
		Want bool
	}{
		{"", "", true},
		{"foo", "", false},
		{"", "bar", false},
		{"foo", "foo", true},
		{"foo", "bar", false},
		{"http://foo:1234", "http://foo:1234", true},
		{"http://foo:1234", "http://foo:5678", true},
		{"http://foo:1234", "http://bar:1234", false},
		{"http://foo:1234", "http://bar:5678", false},
		{"http://foo:80", "http://foo:22", false},
		{"http://foo:80", "http://foo:8080", false},
		{"Hello world:9876!", "Hello world:9876!", true},
		{"Hello world:9876!", "Hello world:1234!", true},
		{
			"Failed to connect to 192.168.1.2:1234: Connection refused",
			"Failed to connect to 192.168.1.2:5678: Connection refused",
			true,
		},
		{
			"Failed to connect to [::1]:1234: Connection refused",
			"Failed to connect to [::1]:5678: Connection refused",
			true,
		},
	}

	for _, tt := range tests {
		if got := isSameIncidentMessage(tt.A, tt.B); got != tt.Want {
			t.Errorf("Expected %v but got %v\nA = %q\nB = %q", tt.Want, got, tt.A, tt.B)
		}
	}
}
