package alert

import (
	"context"
	"errors"
	"sync"

	"github.com/macrat/ayd/internal/ayderr"
	"github.com/macrat/ayd/internal/scheme"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidAlertURL = errors.New("invalid alert URL")
)

// AlertSet is a set of alerts.
// It also implements Alert alertinterface.
type AlertSet []Alert

func NewSet(targets []string) (AlertSet, error) {
	alerts := make(AlertSet, len(targets))
	errs := &ayderr.ListBuilder{What: ErrInvalidAlertURL}

	for i, t := range targets {
		var err error
		alerts[i], err = New(t)
		if err != nil {
			errs.Pushf("%s: %w", t, err)
		}
	}

	return alerts, errs.Build()
}

// Trigger of AlertSet calls all Trigger methods of children parallelly.
// This method blocks until all alerts done.
func (as AlertSet) Trigger(ctx context.Context, lastRecord api.Record, r scheme.Reporter) {
	wg := &sync.WaitGroup{}

	for _, a := range as {
		wg.Add(1)
		go func(a Alert) {
			a.Trigger(ctx, lastRecord, r)
			wg.Done()
		}(a)
	}

	wg.Wait()
}
