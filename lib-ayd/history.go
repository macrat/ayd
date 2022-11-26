package ayd

import (
	"encoding/json"
	"sort"
	"time"
)

// ProbeHistory is the status history data of single target
//
// Deprecated: this struct will removed in future version.
type ProbeHistory struct {
	Target *URL

	// Status is the latest status of the target
	Status Status

	// Status is the same as Time of the latest History record
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

	target, err := ParseURL(jh.Target)
	if err != nil {
		return err
	}

	updated, err := ParseTime(jh.Updated)
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

// byLatestStatus implements sort.Interface for ProbeHistory.
type byLatestStatus []ProbeHistory

func (xs byLatestStatus) Len() int {
	return len(xs)
}

func (xs byLatestStatus) Less(i, j int) bool {
	switch {
	case len(xs[i].Records) == 0 && len(xs[j].Records) > 0:
		return false
	case len(xs[i].Records) > 0 && len(xs[j].Records) == 0:
		return true
	case xs[i].Status != xs[j].Status:
		return xs[i].Status < xs[j].Status
	default:
		return xs[i].Target.String() < xs[j].Target.String()
	}
}

func (xs byLatestStatus) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

// SortProbeHistories sorts list of ProbeHistory by latest status and target URL.
//
// This function will edit slice directly.
//
// Deprecated: this struct will removed in future version.
func SortProbeHistories(hs []ProbeHistory) {
	sort.Sort(byLatestStatus(hs))
}
