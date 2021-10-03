package ayd

import (
	"encoding/json"
	"net/url"
	"time"
)

// Report is a report from Ayd server.
type Report struct {
	// ProbeHistory is the map of ProbeHistory.
	// The key is target URL string, and the value is struct ProbeHistory.
	ProbeHistory map[string]ProbeHistory

	// CurrentIncidents is the list of Incident that current causing.
	CurrentIncidents []Incident

	// IncidentHistory is the list of Incident that already resolved.
	//
	// If you want get current causing incidents, please use CurrentIncidents.
	IncidentHistory []Incident

	// ReportedAt is the time the report created in server.
	ReportedAt time.Time
}

type jsonReport struct {
	ProbeHistory     []ProbeHistory `json:"probe_history"`
	CurrentIncidents []Incident     `json:"current_incidents"`
	IncidentHistory  []Incident     `json:"incident_history"`
	ReportedAt       string         `json:"reported_at"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Report) UnmarshalJSON(data []byte) error {
	var jr jsonReport

	if err := json.Unmarshal(data, &jr); err != nil {
		return err
	}

	reportedAt, err := time.Parse(time.RFC3339, jr.ReportedAt)
	if err != nil {
		return err
	}

	probeHistory := make(map[string]ProbeHistory)
	for _, x := range jr.ProbeHistory {
		probeHistory[x.Target.String()] = x
	}

	*r = Report{
		ProbeHistory:     probeHistory,
		CurrentIncidents: jr.CurrentIncidents,
		IncidentHistory:  jr.IncidentHistory,
		ReportedAt:       reportedAt,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (r Report) MarshalJSON() ([]byte, error) {
	probeHistory := make([]ProbeHistory, 0, len(r.ProbeHistory))
	for _, x := range r.ProbeHistory {
		probeHistory = append(probeHistory, x)
	}

	currentIncidents := make([]Incident, 0, len(r.CurrentIncidents))
	for _, x := range r.CurrentIncidents {
		currentIncidents = append(currentIncidents, x)
	}

	return json.Marshal(jsonReport{
		ProbeHistory:     probeHistory,
		CurrentIncidents: currentIncidents,
		IncidentHistory:  r.IncidentHistory,
		ReportedAt:       r.ReportedAt.Format(time.RFC3339),
	})
}

// TargetURLs returns target URLs that to status checking
func (r Report) TargetURLs() []*url.URL {
	us := make([]*url.URL, len(r.ProbeHistory))

	i := 0
	for _, x := range r.ProbeHistory {
		us[i] = x.Target
		i++
	}

	return us
}
