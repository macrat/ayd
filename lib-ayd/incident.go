package ayd

import (
	"net/url"
	"time"
)

// Incident is a period of failure or unknown status that has the same status and message
type Incident struct {
	Target *url.URL

	Status Status

	Message string

	// CausedAt is the first detected time the target is unhealthy status
	CausedAt time.Time

	// ResolvedAt is the earliest time that detected the target back to healthy status
	ResolvedAt time.Time
}
