package ayd

import (
	"errors"
	"fmt"
)

// The errors in Ayd library can check the error type via errors.Is function.
var (
	// ErrInvalidArgument is a error for if the argument was wrong.
	ErrInvalidArgumentValue = errors.New("invalid argument value")

	// ErrArgumentCount is a error for if the count of arguments was wrong.
	ErrArgumentCount = errors.New("unexpected count of arguments")

	// ErrCommunicate is a error for if connect or communicate with the Ayd server.
	ErrCommunicate = errors.New("server communication error")

	// ErrInvalidRecord is a error for if failed to parse log because it was invalid format.
	ErrInvalidRecord = errors.New("invalid record")

	// ErrIO is a error for if failed to wread/write log.
	ErrIO = errors.New("failed to read/write log")

	// ErrEmptyTarget is a error for if the target URL of Record was empty.
	ErrEmptyTarget = errors.New("invalid record: the target URL is required")
)

// AydError is the error type of Ayd.
//
// Please use errors.Is or errors.Unwrap if you want to know what kind of error is it.
type AydError struct {
	kind    error
	from    error
	message string
}

func newError(kind error, from error, format string, args ...interface{}) AydError {
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
