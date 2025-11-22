package endpoint

import (
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type Store interface {
	// Name returns the Ayd instance name.
	Name() string

	// Targets returns target URLs include inactive target.
	Targets() []string

	// ProbeHistory returns a slice of ProbeHistory.
	ProbeHistory() []api.ProbeHistory

	// CurrentIncidents returns a slice of current incidents.
	CurrentIncidents() []*api.Incident

	// IncidentHistory returns a slice of past incidents.
	IncidentHistory() []*api.Incident

	// MakeReport creates ayd.Report for exporting for endpoint.
	MakeReport(probeHistoryLength int) api.Report

	// ReportInternalError reports Ayd internal error.
	ReportInternalError(scope, message string)

	// Errors returns a list of internal (critical) errors.
	Errors() (healthy bool, messages []string)

	// IncidentCount returns the count of incident causes.
	IncidentCount() int

	// OpenLog opens ayd.LogScanner.
	OpenLog(since, until time.Time) (api.LogScanner, error)
}
