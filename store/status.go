package store

const (
	STATUS_UNKNOWN Status = iota
	STATUS_HEALTHY
	STATUS_FAILURE
)

type Status int8

func ParseStatus(s string) Status {
	switch s {
	case "HEALTHY":
		return STATUS_HEALTHY
	case "FAILURE":
		return STATUS_FAILURE
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
	default:
		return "UNKNOWN"
	}
}
