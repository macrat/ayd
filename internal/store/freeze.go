package store

import (
	"time"

	"github.com/macrat/ayd/internal/store/freeze"
	api "github.com/macrat/ayd/lib-ayd"
)

func freezeProbeHistory(h *ProbeHistory) freeze.ProbeHistory {
	hs := make([]freeze.Record, PROBE_HISTORY_LEN)
	offset := PROBE_HISTORY_LEN - len(h.Records)
	for i := 0; i < offset; i++ {
		hs[i] = freeze.Record{
			Status: "NO_DATA",
		}
	}
	for i, x := range h.Records {
		hs[offset+i] = freeze.Record{
			CheckedAt: x.CheckedAt.Format(time.RFC3339),
			Status:    x.Status.String(),
			Message:   x.Message,
			Latency:   float64(x.Latency.Microseconds()) / 1000,
		}
	}

	status := "NO_DATA"
	updated := ""
	if len(h.Records) > 0 {
		last := h.Records[len(h.Records)-1]
		status = last.Status.String()
		updated = last.CheckedAt.Format(time.RFC3339)
	}

	return freeze.ProbeHistory{
		Target:  h.Target.String(),
		Status:  status,
		History: hs,
		Updated: updated,
	}
}

func freezeIncident(i *api.Incident) freeze.Incident {
	ji := freeze.Incident{
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

func (s *Store) Freeze() freeze.Status {
	ph := s.ProbeHistory()
	ci := s.CurrentIncidents()
	ih := s.IncidentHistory()

	status := freeze.Status{
		CurrentStatus:    make([]freeze.ProbeHistory, len(ph)),
		CurrentIncidents: make([]freeze.Incident, len(ci)),
		IncidentHistory:  make([]freeze.Incident, len(ih)),
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
