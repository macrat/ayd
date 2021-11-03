package endpoint

import (
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

type Store interface {
	// Path returns path to log file.
	Path() string

	// Targets returns target URLs include inactive target.
	Targets() []string

	// ProbeHistory returns a slice of ProbeHistory.
	//
	// XXX: Can I remove direct import store package?
	ProbeHistory() []*store.ProbeHistory

	// MakeReport creates ayd.Report for exporting for endpoint.
	MakeReport() api.Report

	// ReportInternalError reports Ayd internal error.
	ReportInternalError(scope, message string)

	// Errors returns a list of internal (critical) errors.
	Errors() (healthy bool, messages []string)

	// IncidentCount returns the count of incident causes.
	IncidentCount() int
}
