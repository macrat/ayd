package store

import (
	"net/url"
	"time"

	"github.com/macrat/ayd/probe"
)

type Incident struct {
	Target     *url.URL
	Status     probe.Status
	Message    string
	CausedAt   time.Time
	ResolvedAt time.Time
}

func NewIncident(r probe.Result) *Incident {
	return &Incident{
		Target:   r.Target,
		Status:   r.Status,
		Message:  r.Message,
		CausedAt: r.CheckedAt,
	}
}

func (i *Incident) SameTarget(r probe.Result) bool {
	return i.Target.String() == r.Target.String()
}

func (i *Incident) IsContinued(r probe.Result) bool {
	return i.ResolvedAt.IsZero() && i.Status == r.Status && i.Message == r.Message
}
