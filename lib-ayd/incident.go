package ayd

import (
	"time"

	"github.com/goccy/go-json"
)

// Incident is a period of failure or unknown status that has the same status and message
//
// Deprecated: this struct will removed in future version.
type Incident struct {
	Target *URL

	Status Status

	Message string

	// StartsAt is the first detected time the target is unhealthy status
	StartsAt time.Time

	// EndsAt is the earliest time that detected the target back to healthy status
	EndsAt time.Time
}

type jsonIncident struct {
	Target   string `json:"target"`
	Status   Status `json:"status"`
	Message  string `json:"message"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at,omitempty"`
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

	startsAt, err := ParseTime(ji.StartsAt)
	if err != nil {
		return err
	}

	var endsAt time.Time
	if ji.EndsAt != "" {
		endsAt, err = ParseTime(ji.EndsAt)
		if err != nil {
			return err
		}
	}

	*i = Incident{
		Target:   target,
		Status:   ji.Status,
		Message:  ji.Message,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (i Incident) MarshalJSON() ([]byte, error) {
	var endsAt string
	if !i.EndsAt.IsZero() {
		endsAt = i.EndsAt.Format(time.RFC3339)
	}

	return json.Marshal(jsonIncident{
		Target:   i.Target.String(),
		Status:   i.Status,
		Message:  i.Message,
		StartsAt: i.StartsAt.Format(time.RFC3339),
		EndsAt:   endsAt,
	})
}
