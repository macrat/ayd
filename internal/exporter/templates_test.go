package exporter

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/store"
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
	f := templateFuncs["pad_records"].(func([]api.Record) []struct{})

	tests := []struct {
		Input  int
		Output int
	}{
		{0, store.PROBE_HISTORY_LEN},
		{3, store.PROBE_HISTORY_LEN - 3},
		{store.PROBE_HISTORY_LEN + 5, 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%d", tt.Input), func(t *testing.T) {
			result := f(make([]api.Record, tt.Input))
			if tt.Output != len(result) {
				t.Errorf("expected array length is %d but got %d", tt.Output, len(result))
			}
		})
	}
}

func TestURLUnescape(t *testing.T) {
	f := templateFuncs["url_unescape"].(func(u *url.URL) string)

	tests := []struct {
		Input  url.URL
		Output string
	}{
		{
			url.URL{Scheme: "dummy", Fragment: "Aaあ亜"},
			"dummy:#Aaあ亜",
		},
		{
			url.URL{Scheme: "https", Host: "テスト.com", RawQuery: "あ=亜"},
			"https://テスト.com?あ=亜",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input.String(), func(t *testing.T) {
			result := f(&tt.Input)
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
			"with-nodata",
			[]int{1, 2, 3, 5, 5},
			"M35,1 35,0.8 35.5,0.8 36.5,0.6 37.5,0.4 38.5,0 39.5,0 h0.5V1",
		},
		{
			"without-nodata",
			[]int{
				1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0,
				1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0,
			},
			"M0,1 0,0.8888888888888888 0.5,0.8888888888888888 1.5,0.7777777777777778 2.5,0.6666666666666666 3.5,0.5555555555555556 4.5,0.4444444444444444 5.5,0.33333333333333326 6.5,0.2222222222222222 7.5,0.11111111111111105 8.5,0 9.5,1 10.5,0.8888888888888888 11.5,0.7777777777777778 12.5,0.6666666666666666 13.5,0.5555555555555556 14.5,0.4444444444444444 15.5,0.33333333333333326 16.5,0.2222222222222222 17.5,0.11111111111111105 18.5,0 19.5,1 20.5,0.8888888888888888 21.5,0.7777777777777778 22.5,0.6666666666666666 23.5,0.5555555555555556 24.5,0.4444444444444444 25.5,0.33333333333333326 26.5,0.2222222222222222 27.5,0.11111111111111105 28.5,0 29.5,1 30.5,0.8888888888888888 31.5,0.7777777777777778 32.5,0.6666666666666666 33.5,0.5555555555555556 34.5,0.4444444444444444 35.5,0.33333333333333326 36.5,0.2222222222222222 37.5,0.11111111111111105 38.5,0 39.5,1 h0.5V1",
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
