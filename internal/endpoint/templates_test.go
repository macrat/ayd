package endpoint

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestInvertIncidents(t *testing.T) {
	f := templateFuncs["invert_incidents"].(func([]api.Incident) []api.Incident)

	t.Run("no_incidents", func(t *testing.T) {
		result := f([]api.Incident{})
		if len(result) != 0 {
			t.Fatalf("unexpected result length: %d", len(result))
		}
	})

	t.Run("three_incidents", func(t *testing.T) {
		input := []api.Incident{
			{Message: "foo"},
			{Message: "bar"},
			{Message: "baz"},
		}
		expect := []api.Incident{
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

func TestPadRecords(t *testing.T) {
	f := templateFuncs["pad_records"].(func(int, []api.Record) []struct{})

	tests := []struct {
		Length  int
		Records int
		Output  int
	}{
		{40, 0, 40},
		{40, 3, 37},
		{40, 40, 0},
		{20, 40, 0},
		{20, 10, 10},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%d_%d", tt.Length, tt.Records), func(t *testing.T) {
			result := f(tt.Length, make([]api.Record, tt.Records))
			if tt.Output != len(result) {
				t.Errorf("expected array length is %d but got %d", tt.Output, len(result))
			}
		})
	}
}

type DummyStringer string

func (s DummyStringer) String() string {
	return string(s)
}

func TestToCamel(t *testing.T) {
	f := templateFuncs["to_camel"].(func(s fmt.Stringer) string)

	tests := []struct {
		Input  DummyStringer
		Output string
	}{
		{"hello", "Hello"},
		{"WORLD", "World"},
		{"FooBar", "Foobar"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input.String(), func(t *testing.T) {
			result := f(tt.Input)
			if tt.Output != result {
				t.Errorf("expected output is %s but got %s", tt.Output, result)
			}
		})
	}
}

func TestLatencyGraph(t *testing.T) {
	f := templateFuncs["latency_graph"].(func(rs []api.Record) string)

	tests := []struct {
		Name   string
		Input  []int
		Output string
	}{
		{
			"empty",
			[]int{},
			"",
		},
		{
			"with-nodata",
			[]int{1, 2, 3, 5, 5},
			"M15,1 15,0.8 15.5,0.8 16.5,0.6 17.5,0.4 18.5,0 19.5,0 h0.5V1",
		},
		{
			"without-nodata",
			[]int{
				1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0,
			},
			"M0,1 0,0.8888888888888888 0.5,0.8888888888888888 1.5,0.7777777777777778 2.5,0.6666666666666666 3.5,0.5555555555555556 4.5,0.4444444444444444 5.5,0.33333333333333326 6.5,0.2222222222222222 7.5,0.11111111111111105 8.5,0 9.5,1 10.5,0.8888888888888888 11.5,0.7777777777777778 12.5,0.6666666666666666 13.5,0.5555555555555556 14.5,0.4444444444444444 15.5,0.33333333333333326 16.5,0.2222222222222222 17.5,0.11111111111111105 18.5,0 19.5,1 h0.5V1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			input := make([]api.Record, len(tt.Input))
			for i, latency := range tt.Input {
				input[i].Latency = time.Duration(latency)
			}

			result := f(input)

			if tt.Output != result {
				t.Errorf("unexpected output\nexpected: %#v\n but got: %#v", tt.Output, result)
			}
		})
	}
}
