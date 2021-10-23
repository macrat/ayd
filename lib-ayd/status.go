package ayd

const (
	// StatusUnknown means UNKNOWN current status because failed to check the target status.
	// System administrators have to fix Ayd settings, or do something to the target system, when this status.
	StatusUnknown Status = iota

	// StatusHealthy means success to status check and the target is HEALTHY.
	StatusHealthy

	// StatusDebased means success to status check and the target is worked but partially features or stability is DEBASED.
	// System administrators have to do something action to the target system when this status, but might not urgency.
	StatusDebased

	// StatusFailure means the target is in FAILURE, but status check is success.
	// System administrators have to do something action to the target system when this status.
	StatusFailure

	// StatusAborted means the status check ABORTED because stop by system administrator or other system program like systemd.
	// System administrators don't have to do something on this status.
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
	case "DEBASED":
		return StatusDebased
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
	case StatusDebased:
		return "DEBASED"
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
