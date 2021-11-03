package url

import (
	"net/url"

	api "github.com/macrat/ayd/lib-ayd"
)

type Reporter interface {
	// Report reports a Record.
	//
	// `source` in argument is the probe's URL.
	Report(source *url.URL, r api.Record)

	// DeactivateTarget marks the target is no longer reported via specified source.
	DeactivateTarget(source *url.URL, targets ...*url.URL)
}
