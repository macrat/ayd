package probe

import (
	"net/url"
	"time"
)

type Status int8

func ParseStatus(s string) Status {
	switch s {
	case "OK":
		return STATUS_OK
	case "FAIL":
		return STATUS_FAIL
	default:
		return STATUS_UNKNOWN
	}
}

func (s Status) String() string {
	switch s {
	case STATUS_OK:
		return "OK"
	case STATUS_FAIL:
		return "FAIL"
	default:
		return "????"
	}
}

const (
	STATUS_UNKNOWN Status = iota
	STATUS_OK
	STATUS_FAIL
)

type Result struct {
	CheckedAt time.Time
	Target    *url.URL
	Status    Status
	Message   string
	Latency   time.Duration
}
