package ayd

import (
	"errors"
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
