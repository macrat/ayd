package probe

import (
	"reflect"
	"testing"
)

func TestAutoDecode(t *testing.T) {
	tests := []struct {
		Name   string
		Input  []byte
		Output []byte
	}{
		{"あ", []byte{0xEF, 0xBB, 0xBF, 0xE3, 0x81, 0x82}, []byte{0xE3, 0x81, 0x82}},
		{"文", []byte{0xEF, 0xBB, 0xBF, 0xE6, 0x96, 0x87, 0x0A}, []byte{0xE6, 0x96, 0x87, 0x0A}},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output := autoDecode(tt.Input)
			if reflect.DeepEqual(output, tt.Output) {
				t.Errorf("expected %#v but got %#v", tt.Output, output)
			}
		})
	}
}
