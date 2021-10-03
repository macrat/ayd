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

	// Status is the same as CheckedAt of the latest History record
	Updated time.Time

	Records []Record
}

type jsonProbeHistory struct {
	Target  string   `json:"target"`
	Status  Status   `json:"status"`
	Updated string   `json:"updated,omitempty"`
	Records []Record `json:"records"`
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
		Records: jh.Records,
		Updated: updated,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (ph ProbeHistory) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonProbeHistory{
		Target:  ph.Target.String(),
		Status:  ph.Status,
		Records: ph.Records,
		Updated: ph.Updated.Format(time.RFC3339),
	})
}
