package ayd

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidArgument is a error for if the argument was wrong.
	//
	// Please use with errors.Is function.
	ErrInvalidArgumentValue = errors.New("invalid argument value")

	// ErrArgumentCount is a error for if the count of arguments was wrong.
	//
	// Please use with errors.Is function.
	ErrArgumentCount = errors.New("unexpected count of arguments")

	// ErrCommunicate is a error for if connect or communicate with the Ayd server.
	//
	// Please use with errors.Is function.
	ErrCommunicate = errors.New("server communication error")

	// ErrInvalidRecord is a error for if failed to parse log because it was invalid format.
	//
	// Please use with errors.Is function.
	ErrInvalidRecord = errors.New("invalid record")

	// ErrIO is a error for if failed to wread/write log.
	//
	// Please use with errors.Is function.
	ErrIO = errors.New("failed to read/write log")

	// ErrEmptyTarget is a error for if the target URL of Record was empty.
	//
	// Please use with errors.Is function.
	ErrEmptyTarget = errors.New("invalid record: the target URL is required")
)

// AydError is the error type of Ayd.
//
// You can use errors.Is and errors.Unwrap to this type.
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
