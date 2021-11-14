package endpoint

import (
	"strings"
	"testing"
)

func TestBinSearch(t *testing.T) {
	tests := []struct {
		Input  string
		Expect int
	}{
		{"01234-678#01+345+789", 6},
		{"01234-678#01+345+78", 6},
		{"0123-567#90+234+67", 5},
		{"0-23#56+890+23+56+8", 2},
		{"0-234-6789-123-567#90", 15},
		{"012345678901234567890", 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Input, func(t *testing.T) {
			result, _ := binSearch(len(tt.Input), func(pos int) (isAfter bool, endPos int, err error) {
				t.Log(tt.Input)
				t.Log(strings.Repeat(" ", pos) + "^")
				for i, x := range tt.Input[pos:] {
					switch x {
					case '-':
						return false, i + pos, nil
					case '+', '#':
						return true, i + pos, nil
					}
				}
				return true, len(tt.Input) - 1, nil
			})

			if result != tt.Expect {
				t.Fatalf("expected %d but got %d", tt.Expect, result)
			}
		})
	}
}
