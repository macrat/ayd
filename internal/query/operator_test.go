package query

import (
	"testing"
)

func TestOperator_Invert(t *testing.T) {
	tests := []struct {
		input  operator
		output operator
		ok     bool
	}{
		{opIncludes, 0, false},             // There is no operator for "not in"
		{opEqual, opNotEqual, true},        // = -> !=
		{opNotEqual, opEqual, true},        // != -> =
		{opLessThan, opGreaterEqual, true}, // < -> >=
		{opGreaterThan, opLessEqual, true}, // > -> <=
		{opLessEqual, opGreaterThan, true}, // <= -> >
		{opGreaterEqual, opLessThan, true}, // >= -> <
	}
	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result, ok := tt.input.Invert()
			if result != tt.output || ok != tt.ok {
				t.Errorf("expected (%v, %v), got (%v, %v)", tt.output, tt.ok, result, ok)
			}
		})
	}
}
