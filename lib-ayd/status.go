package ayd

const (
	// StatusUnknown means UNKNOWN current status because failed to check the target status
	StatusUnknown Status = iota

	// StatusHealthy means success to status check and the target is HEALTHY
	StatusHealthy

	// StatusFailure means the target is in FAILURE, but status check is success
	StatusFailure

	// StatusAborted means the status check ABORTED because stop Ayd server
	StatusAborted
)

// Status is the status of target service
type Status int8

// ParseStatus is parse status string
//
// If passed unsupported status, it will returns StatusUnknown
func ParseStatus(raw string) Status {
	switch raw {
	case "HEALTHY":
		return StatusHealthy
	case "FAILURE":
		return StatusFailure
	case "ABORTED":
		return StatusAborted
	default:
		return StatusUnknown
	}
}

// UnmarshalText is unmarshal text as status
//
// This function always returns nil.
// This parses as StatusUnknown instead of returns error if unsupported status passed.
func (s *Status) UnmarshalText(text []byte) error {
	*s = ParseStatus(string(text))
	return nil
}

// String is make Status a string
func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "HEALTHY"
	case StatusFailure:
		return "FAILURE"
	case StatusAborted:
		return "ABORTED"
	default:
		return "UNKNOWN"
	}
}

// MarshalText is marshal Status as text
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}
