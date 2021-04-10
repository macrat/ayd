package exporter

import (
	"time"

	"github.com/macrat/ayd/store"
)

type frozenProbeHistory struct {
	Target  string `json:"target"`
	Status  string `json:"status"`
	History string `json:"history"`
	Updated string `json:"updated"`
}

func freezeProbeHistory(h *store.ProbeHistory) frozenProbeHistory {
	hs := ""
	for _, x := range h.Results {
		switch x.Status {
		case store.STATUS_OK:
			hs += "O"
		case store.STATUS_FAIL:
			hs += "F"
		default:
			hs += "?"
		}
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
