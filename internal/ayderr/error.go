package ayderr

import (
	"fmt"
)

// AydError is the error type of Ayd.
//
// Please use errors.Is or errors.Unwrap if you want to know what kind of error is it.
type AydError struct {
	kind    error
	from    error
	message string
}

// New creates a new AydError.
func New(kind error, from error, format string, args ...interface{}) AydError {
	msg := fmt.Sprintf(format, args...)
	if from != nil {
		if msg != "" {
			msg += ": "
		}
		msg += from.Error()
	}

	return AydError{
		kind:    kind,
		from:    from,
		message: msg,
	}
}

// Error implements error interface.
func (e AydError) Error() string {
	return e.message
}

// Unwrap implement for errors.Unwrap.
func (e AydError) Unwrap() error {
	return e.from
}

// Is implement for errors.Is.
func (e AydError) Is(err error) bool {
	return e.kind == err
}
