package ayd

import (
	"encoding/json"
	"time"
)

// Incident is a period of failure or unknown status that has the same status and message
//
// Deprecated: this struct will removed in future version.
type Incident struct {
	Target *URL

	Status Status

	Message string

	// CausedAt is the first detected time the target is unhealthy status
	CausedAt time.Time

	// ResolvedAt is the earliest time that detected the target back to healthy status
	ResolvedAt time.Time
}

type jsonIncident struct {
	Target     string `json:"target"`
	Status     Status `json:"status"`
	Message    string `json:"message"`
	CausedAt   string `json:"caused_at"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (i *Incident) UnmarshalJSON(data []byte) error {
	var ji jsonIncident

	if err := json.Unmarshal(data, &ji); err != nil {
		return err
	}

	target, err := ParseURL(ji.Target)
	if err != nil {
		return err
	}

	causedAt, err := time.Parse(time.RFC3339, ji.CausedAt)
	if err != nil {
		return err
	}

	var resolvedAt time.Time
	if ji.ResolvedAt != "" {
		resolvedAt, err = time.Parse(time.RFC3339, ji.ResolvedAt)
		if err != nil {
			return err
		}
	}

	*i = Incident{
		Target:     target,
		Status:     ji.Status,
		Message:    ji.Message,
		CausedAt:   causedAt,
		ResolvedAt: resolvedAt,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (i Incident) MarshalJSON() ([]byte, error) {
	var resolvedAt string
	if !i.ResolvedAt.IsZero() {
		resolvedAt = i.ResolvedAt.Format(time.RFC3339)
	}

	return json.Marshal(jsonIncident{
		Target:     i.Target.String(),
		Status:     i.Status,
		Message:    i.Message,
		CausedAt:   i.CausedAt.Format(time.RFC3339),
		ResolvedAt: resolvedAt,
	})
}
