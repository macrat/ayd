package exporter

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"

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
