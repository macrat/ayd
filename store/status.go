package store

const (
	STATUS_UNKNOWN Status = iota
	STATUS_HEALTHY
	STATUS_FAILURE
	STATUS_ABORTED
)

type Status int8

func ParseStatus(s string) Status {
	switch s {
	case "HEALTHY":
		return STATUS_HEALTHY
	case "FAILURE":
		return STATUS_FAILURE
	case "ABORTED":
		return STATUS_ABORTED
	default:
		return STATUS_UNKNOWN
	}
}

func (s Status) String() string {
	switch s {
	case STATUS_HEALTHY:
		return "HEALTHY"
	case STATUS_FAILURE:
		return "FAILURE"
	case STATUS_ABORTED:
		return "ABORTED"
	default:
		return "UNKNOWN"
	}
}
