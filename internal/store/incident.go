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
		StartsAt: r.Time,
	}
}

type byIncidentCaused []*api.Incident

func (xs byIncidentCaused) Len() int {
	return len(xs)
}

func (xs byIncidentCaused) Less(i, j int) bool {
	if xs[i].StartsAt.Equal(xs[j].StartsAt) {
		return xs[i].Target.String() < xs[j].Target.String()
	}
	return xs[i].StartsAt.Before(xs[j].StartsAt)
}

func (xs byIncidentCaused) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}
