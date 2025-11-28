package mcp

import (
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

// Store is an interface for accessing Ayd monitoring data.
type Store interface {
	// Name returns the Ayd instance name.
	Name() string

	// ProbeHistory returns a slice of ProbeHistory.
	ProbeHistory() []api.ProbeHistory

	// CurrentIncidents returns a slice of current incidents.
	CurrentIncidents() []*api.Incident

	// IncidentHistory returns a slice of past incidents.
	IncidentHistory() []*api.Incident

	// ReportInternalError reports Ayd internal error.
	ReportInternalError(scope, message string)

	// OpenLog opens api.LogScanner.
	OpenLog(since, until time.Time) (api.LogScanner, error)
}
