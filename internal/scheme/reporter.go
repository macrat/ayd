package scheme

import (
	api "github.com/macrat/ayd/lib-ayd"
)

type Reporter interface {
	// Report reports a Record.
	//
	// `source` in argument is the probe's URL.
	Report(source *api.URL, r api.Record)

	// DeactivateTarget marks the target is no longer reported via specified source.
	DeactivateTarget(source *api.URL, targets ...*api.URL)
}
