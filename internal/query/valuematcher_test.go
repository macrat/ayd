package query

import (
	"fmt"
	"testing"
	"time"
)

func TestParseTimeValueMatcher(t *testing.T) {
	today := time.Now().Format("2006-01-02")

	tests := []struct {
		opCode operator
		input  string
		want   string
	}{
		{opEqual, "2024-01-02T15:04:05Z", "2024-01-02T15:04:05.0000Z"},
		{opEqual, "2024-01-02T15:04:05.1234Z", "2024-01-02T15:04:05.1234Z"},
		{opEqual, "2024-01-02T15:04Z", "2024-01-02T15:04:00.0000Z"},
		{opEqual, "2024-01-02T15Z", "2024-01-02T15:00:00.0000Z"},
		{opEqual, "2024-01-02T15:04:05", "2024-01-02T15:04:05.0000Z"},
		{opEqual, "2024-01-02T15:04:05.1234", "2024-01-02T15:04:05.1234Z"},
		{opEqual, "2024-01-02T15:04", "2024-01-02T15:04:00.0000Z"},
		{opEqual, "2024-01-02T15", "2024-01-02T15:00:00.0000Z"},
		{opEqual, "2024-01-02", "2024-01-02T00:00:00.0000Z"},
		{opEqual, "2024-01-02 15:04", "2024-01-02T15:04:00.0000Z"},
		{opLessThan, "2024-01-02", "2024-01-02T00:00:00.0000Z"},
		{opLessEqual, "2024-01-02", "2024-01-02T23:59:59.9999Z"},
		{opGreaterThan, "2024-01-02", "2024-01-02T23:59:59.9999Z"},
		{opGreaterEqual, "2024-01-02", "2024-01-02T00:00:00.0000Z"},
		{opEqual, "16:05:07.1234", today + "T16:05:07.1234Z"},
		{opEqual, "16:05:07", today + "T16:05:07.0000Z"},
		{opEqual, "16:05", today + "T16:05:00.0000Z"},
		{opLessThan, "16:05", today + "T16:05:00.0000Z"},
		{opLessEqual, "16:05", today + "T16:05:59.9999Z"},
		{opGreaterThan, "16:05", today + "T16:05:59.9999Z"},
		{opGreaterEqual, "16:05", today + "T16:05:00.0000Z"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s%s", test.opCode, test.input), func(t *testing.T) {
			m, err := parseTimeValueMatcher(test.opCode, test.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := m.Value.Format("2006-01-02T15:04:05.0000Z"); got != test.want {
				t.Errorf("unexpected result:\ngot  %s\nwant %s", got, test.want)
			}
		})
	}
}

func FuzzParseTimeValueMatcher(f *testing.F) {
	f.Add(uint8(opEqual), "2024-01-02T15:04:05Z")
	f.Add(uint8(opLessThan), "2024-01-02T15:04:05")
	f.Add(uint8(opGreaterThan), "2024-01-02T15:04Z")
	f.Add(uint8(opLessEqual), "2024-01-02 15:04:05")
	f.Add(uint8(opGreaterEqual), "2024-01-02_01:02:03.4567")
	f.Add(uint8(opNotEqual), "2024-01-02T15:04Z")
	f.Add(uint8(opIncludes), "2024-01-02")
	f.Add(uint8(opIncludes), "15:04:05")
	f.Add(uint8(opIncludes), "15:04")

	f.Fuzz(func(t *testing.T, opCode uint8, input string) {
		switch operator(opCode) {
		case opIncludes, opEqual, opLessThan, opGreaterThan, opLessEqual, opGreaterEqual, opNotEqual:
		default:
			t.Skip()
		}

		_, err := parseTimeValueMatcher(operator(opCode), input)
		if err != nil {
			t.Skip()
		}
	})
}
