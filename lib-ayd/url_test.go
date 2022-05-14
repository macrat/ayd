package ayd_test

import (
	"testing"

	"github.com/macrat/ayd/lib-ayd"
)

func TestURLUnescape(t *testing.T) {
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
