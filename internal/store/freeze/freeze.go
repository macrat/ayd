package freeze

type Record struct {
	CheckedAt string  `json:"checked_at,omitempty"`
	Status    string  `json:"status"`
	Message   string  `json:"message"`
	Latency   float64 `json:"latency"`
}

type ProbeHistory struct {
	Target  string   `json:"target"`
	Status  string   `json:"status"`
	History []Record `json:"history"`
	Updated string   `json:"updated,omitempty"`
}

type Incident struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	CausedAt   string `json:"caused_at"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

type Status struct {
	CurrentStatus    []ProbeHistory `json:"current_status"`
	CurrentIncidents []Incident     `json:"current_incidents"`
	IncidentHistory  []Incident     `json:"incident_history"`
	ReportedAt       string         `json:"reported_at"`
}
