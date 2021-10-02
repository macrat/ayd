package ayd

import (
	"encoding/json"
	"net/url"
	"time"
)

// ProbeHistory is the status history data of single target
type ProbeHistory struct {
	Target *url.URL

	// Status is the latest status of the target
	Status Status

	History []Record

	// Status is the same as CheckedAt of the latest History record
	Updated time.Time
}

type jsonProbeHistory struct {
	Target  string   `json:"target"`
	Status  Status   `json:"status"`
	History []Record `json:"history"`
	Updated string   `json:"updated,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ph *ProbeHistory) UnmarshalJSON(data []byte) error {
	var jh jsonProbeHistory

	if err := json.Unmarshal(data, &jh); err != nil {
		return err
	}

	target, err := url.Parse(jh.Target)
	if err != nil {
		return err
	}

	updated, err := time.Parse(time.RFC3339, jh.Updated)
	if err != nil {
		return err
	}

	*ph = ProbeHistory{
		Target:  target,
		Status:  jh.Status,
		History: jh.History,
		Updated: updated,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (ph ProbeHistory) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonProbeHistory{
		Target:  ph.Target.String(),
		Status:  ph.Status,
		History: ph.History,
		Updated: ph.Updated.Format(time.RFC3339),
	})
}
