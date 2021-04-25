package store

import (
	"net/url"
	"time"
)

type Incident struct {
	Target     *url.URL
	Status     Status
	Message    string
	CausedAt   time.Time
	ResolvedAt time.Time
}

func NewIncident(r Record) *Incident {
	return &Incident{
		Target:   r.Target,
		Status:   r.Status,
		Message:  r.Message,
		CausedAt: r.CheckedAt,
	}
}

func (i *Incident) SameTarget(r Record) bool {
	return i.Target.String() == r.Target.String()
}

func (i *Incident) IsContinued(r Record) bool {
	return i.ResolvedAt.IsZero() && i.Status == r.Status && i.Message == r.Message
}

type byIncidentCaused []*Incident

func (xs byIncidentCaused) Len() int {
	return len(xs)
}

func (xs byIncidentCaused) Less(i, j int) bool {
	return xs[i].CausedAt.Before(xs[j].CausedAt)
}

func (xs byIncidentCaused) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}
