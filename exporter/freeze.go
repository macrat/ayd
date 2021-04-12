package exporter

import (
	"time"

	"github.com/macrat/ayd/store"
)

type frozenRecord struct {
	CheckedAt string  `json:"checked_at,omitempty"`
	Status    string  `json:"status"`
	Message   string  `json:"message,omitempty"`
	Latency   float64 `json:"latency,omitempty"`
}

type frozenProbeHistory struct {
	Target  string         `json:"target"`
	Status  string         `json:"status"`
	History []frozenRecord `json:"history"`
	Updated string         `json:"updated"`
}

func freezeProbeHistory(h *store.ProbeHistory) frozenProbeHistory {
	hs := []frozenRecord{}
	for i := 0; i < store.PROBE_HISTORY_LEN-len(h.Results); i++ {
		hs = append(hs, frozenRecord{
			Status: "NA",
		})
	}
	for _, x := range h.Results {
		hs = append(hs, frozenRecord{
			CheckedAt: x.CheckedAt.Format(time.RFC3339),
			Status:    x.Status.String(),
			Message:   x.Message,
			Latency:   float64(x.Latency.Microseconds()) / 1000,
		})
	}

	last := h.Results[len(h.Results)-1]

	return frozenProbeHistory{
		Target:  h.Target.String(),
		Status:  last.Status.String(),
		History: hs,
		Updated: last.CheckedAt.Format(time.RFC3339),
	}
}

type frozenIncident struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	CausedAt   string `json:"caused_at"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

func freezeIncident(i *store.Incident) frozenIncident {
	ji := frozenIncident{
		Target:   i.Target.String(),
		Status:   i.Status.String(),
		Message:  i.Message,
		CausedAt: i.CausedAt.Format(time.RFC3339),
	}
	if !i.ResolvedAt.IsZero() {
		ji.ResolvedAt = i.ResolvedAt.Format(time.RFC3339)
	}
	return ji
}

type frozenStatus struct {
	CurrentStatus    []frozenProbeHistory `json:"current_status"`
	CurrentIncidents []frozenIncident     `json:"current_incidents"`
	IncidentHistory  []frozenIncident     `json:"incident_history"`
	ReportedAt       string               `json:"reported_at"`
}

func freezeStatus(s *store.Store) frozenStatus {
	status := frozenStatus{
		CurrentIncidents: []frozenIncident{},
		IncidentHistory:  []frozenIncident{},
		ReportedAt:       time.Now().Format(time.RFC3339),
	}

	for _, r := range s.ProbeHistory.AsSortedArray() {
		status.CurrentStatus = append(status.CurrentStatus, freezeProbeHistory(r))
	}

	for _, i := range s.CurrentIncidents {
		status.CurrentIncidents = append(status.CurrentIncidents, freezeIncident(i))
	}
	for _, i := range s.IncidentHistory {
		status.IncidentHistory = append(status.IncidentHistory, freezeIncident(i))
	}

	return status
}
