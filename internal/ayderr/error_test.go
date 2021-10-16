package ayderr_test

import (
	"errors"
	"testing"

	"github.com/macrat/ayd/internal/ayderr"
	"github.com/macrat/ayd/lib-ayd"
)

func TestAydError(t *testing.T) {
	tests := []struct {
		kind    error
		from    error
		format  string
		args    []interface{}
		message string
	}{
		{
			ayd.ErrInvalidArgumentValue,
			ayd.ErrArgumentCount,
			"hello %s",
			[]interface{}{"world"},
			"hello world: unexpected count of arguments",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.message, func(t *testing.T) {
			err := ayderr.New(tt.kind, tt.from, tt.format, tt.args...)

			if err.Error() != tt.message {
				t.Errorf("unexpected message: %s", err)
			}

			if !errors.Is(err, tt.kind) {
				t.Errorf("error is %#v but reports as not", tt.kind)
			}

			if !errors.Is(err, tt.from) {
				t.Errorf("error is sub error of %#v but reports as not", tt.from)
			}
		})
	}
}
