package store

import (
	api "github.com/macrat/ayd/lib-ayd"
)

// newIncidens makes a new api.Incident from an api.Record.
func newIncident(r api.Record) *api.Incident {
	return &api.Incident{
		Target:   r.Target,
		Status:   r.Status,
		Message:  r.Message,
		CausedAt: r.Time,
	}
}

// incidentIsContinued checks if an incident is stil continued or not.
func incidentIsContinued(i *api.Incident, r api.Record) bool {
	return i.ResolvedAt.IsZero() && i.Status == r.Status && i.Message == r.Message
}

type byIncidentCaused []*api.Incident

func (xs byIncidentCaused) Len() int {
	return len(xs)
}

func (xs byIncidentCaused) Less(i, j int) bool {
	if xs[i].CausedAt.Equal(xs[j].CausedAt) {
		return xs[i].Target.String() < xs[j].Target.String()
	}
	return xs[i].CausedAt.Before(xs[j].CausedAt)
}

func (xs byIncidentCaused) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}
