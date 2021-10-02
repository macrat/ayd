package store

import (
	api "github.com/macrat/ayd/lib-ayd"
)

func NewIncident(r api.Record) *api.Incident {
	return &api.Incident{
		Target:   r.Target,
		Status:   r.Status,
		Message:  r.Message,
		CausedAt: r.CheckedAt,
	}
}

func IncidentIsContinued(i *api.Incident, r api.Record) bool {
	return i.ResolvedAt.IsZero() && i.Status == r.Status && i.Message == r.Message
}

type byIncidentCaused []*api.Incident

func (xs byIncidentCaused) Len() int {
	return len(xs)
}

func (xs byIncidentCaused) Less(i, j int) bool {
	return xs[i].CausedAt.Before(xs[j].CausedAt)
}

func (xs byIncidentCaused) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}
