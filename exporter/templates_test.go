package exporter

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/macrat/ayd/store/freeze"
)

func TestInvertIncidents(t *testing.T) {
	f := templateFuncs["invert_incidents"].(func([]freeze.Incident) []freeze.Incident)

	t.Run("no_incidents", func(t *testing.T) {
		result := f([]freeze.Incident{})
		if len(result) != 0 {
			t.Fatalf("unexpected result length: %d", len(result))
		}
	})

	t.Run("three_incidents", func(t *testing.T) {
		input := []freeze.Incident{
			{Message: "foo"},
			{Message: "bar"},
			{Message: "baz"},
		}
		expect := []freeze.Incident{
			{Message: "baz"},
			{Message: "bar"},
			{Message: "foo"},
		}

		result := f(input)
		if len(result) != len(expect) {
			t.Fatalf("unexpected result length: %d", len(result))
		}

		for i := range result {
			if result[i].Message != expect[i].Message {
				t.Errorf("%d: unexpected message: %#v", i, result[i].Message)
			}
		}
	})
}

func TestBreakText(t *testing.T) {
	f := templateFuncs["break_text"].(func(string, int) []string)

	tests := []struct {
		Input  string
		Width  int
		Output []string
	}{
		{"hello_world", 20, []string{"hello_world"}},
		{"hello_world", 5, []string{"hello", "_worl", "d"}},
		{"foobar", 3, []string{"foo", "bar"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s_%d", tt.Input, tt.Width), func(t *testing.T) {
			result := f(tt.Input, tt.Width)
			if !reflect.DeepEqual(tt.Output, result) {
				t.Errorf("expected %#v\n but got %#v", tt.Output, result)
			}
		})
	}
}

func TestAlignCenter(t *testing.T) {
	f := templateFuncs["align_center"].(func(string, int) string)

	tests := []struct {
		Input  string
		Width  int
		Output string
	}{
		{"foobar", 10, "  foobar"},
		{"foo_bar", 10, " foo_bar"},
		{"foobar", 5, "foobar"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s_%d", tt.Input, tt.Width), func(t *testing.T) {
			result := f(tt.Input, tt.Width)
			if tt.Output != result {
				t.Errorf("expected %#v\n but got %#v", tt.Output, result)
			}
		})
	}
}
