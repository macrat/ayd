package exporter

import (
	"encoding/json"
	"time"
	"net/http"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

type JSONTargetStatus struct {
	Target  string `json:"target"`
	Status  string `json:"status"`
	History string `json:"history"`
	Updated string `json:"updated"`
}

func NewJSONTargetStatus(h *store.ProbeHistory) JSONTargetStatus {
	hs := ""
	for _, x := range h.Results {
		switch x.Status {
		case probe.STATUS_OK:
			hs += "O"
		case probe.STATUS_FAIL:
			hs += "F"
		default:
			hs += "?"
		}
	}

	last := h.Results[len(h.Results)-1]

	return JSONTargetStatus{
		Target: h.Target.String(),
		Status: last.Status.String(),
		History: hs,
		Updated: last.CheckedAt.Format(time.RFC3339),
	}
}

type JSONIncident struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	CausedAt   string `json:"caused_at"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

func NewJSONIncident(i *store.Incident) JSONIncident {
	ji := JSONIncident{
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

type JSONStatus struct {
	CurrentStatus    []JSONTargetStatus `json:"current_status"`
	CurrentIncidents []JSONIncident     `json:"current_incidents"`
	IncidentHistory  []JSONIncident     `json:"incident_history"`
	ReportedAt       string             `json:"reported_at"`
}

func JSONExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		status := JSONStatus{
			CurrentIncidents: []JSONIncident{},
			IncidentHistory: []JSONIncident{},
			ReportedAt: time.Now().Format(time.RFC3339),
		}

		for _, r := range s.ProbeHistory.AsSortedArray() {
			status.CurrentStatus = append(status.CurrentStatus, NewJSONTargetStatus(r))
		}

		for _, i := range s.CurrentIncidents {
			status.CurrentIncidents = append(status.CurrentIncidents, NewJSONIncident(i))
		}
		for _, i := range s.IncidentHistory {
			status.IncidentHistory = append(status.IncidentHistory, NewJSONIncident(i))
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		enc.Encode(status)
	}
}
