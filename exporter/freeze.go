package exporter

import (
	"time"

	"github.com/macrat/ayd/store"
)

type frozenRecord struct {
	CheckedAt  string  `json:"checked_at,omitempty"`
	Status     string  `json:"status"`
	Message    string  `json:"message"`
	Latency    float64 `json:"latency"`
	LatencyStr string  `json:"-"`
}

type frozenProbeHistory struct {
	Target  string         `json:"target"`
	Status  string         `json:"status"`
	History []frozenRecord `json:"history"`
	Updated string         `json:"updated,omitempty"`
}

func freezeProbeHistory(h *store.ProbeHistory) frozenProbeHistory {
	hs := make([]frozenRecord, store.PROBE_HISTORY_LEN)
	offset := store.PROBE_HISTORY_LEN - len(h.Records)
	for i := 0; i < offset; i++ {
		hs[i] = frozenRecord{
			Status: "NO_DATA",
		}
	}
	for i, x := range h.Records {
		hs[offset+i] = frozenRecord{
			CheckedAt:  x.CheckedAt.Format(time.RFC3339),
			Status:     x.Status.String(),
			Message:    x.Message,
			Latency:    float64(x.Latency.Microseconds()) / 1000,
			LatencyStr: x.Latency.String(),
		}
	}

	status := "NO_DATA"
	updated := ""
	if len(h.Records) > 0 {
		last := h.Records[len(h.Records)-1]
		status = last.Status.String()
		updated = last.CheckedAt.Format(time.RFC3339)
	}

	return frozenProbeHistory{
		Target:  h.Target.String(),
		Status:  status,
		History: hs,
		Updated: updated,
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
	ph := s.ProbeHistory()
	ci := s.CurrentIncidents()
	ih := s.IncidentHistory()

	status := frozenStatus{
		CurrentStatus:    make([]frozenProbeHistory, len(ph)),
		CurrentIncidents: make([]frozenIncident, len(ci)),
		IncidentHistory:  make([]frozenIncident, len(ih)),
		ReportedAt:       time.Now().Format(time.RFC3339),
	}

	for i, x := range ph {
		status.CurrentStatus[i] = freezeProbeHistory(x)
	}

	for i, x := range ci {
		status.CurrentIncidents[i] = freezeIncident(x)
	}
	for i, x := range ih {
		status.IncidentHistory[i] = freezeIncident(x)
	}

	return status
}
