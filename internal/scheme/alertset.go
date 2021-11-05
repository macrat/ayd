package scheme

import (
	"context"
	"errors"
	"sync"

	"github.com/macrat/ayd/internal/ayderr"
	api "github.com/macrat/ayd/lib-ayd"
)

var (
	ErrInvalidAlertURL = errors.New("invalid alert URL")
)

// AlertSet is a set of alerts.
// It also implements Alerter alertinterface.
type AlertSet []Alerter

func NewAlertSet(targets []string) (AlertSet, error) {
	alerts := make(AlertSet, len(targets))
	errs := &ayderr.ListBuilder{What: ErrInvalidAlertURL}

	for i, t := range targets {
		var err error
		alerts[i], err = NewAlert(t)
		if err != nil {
			errs.Pushf("%s: %w", t, err)
		}
	}

	return alerts, errs.Build()
}

// Alert of AlertSet calls all Alert methods of children parallelly.
// This method blocks until all alerts done.
func (as AlertSet) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	wg := &sync.WaitGroup{}

	for _, a := range as {
		wg.Add(1)
		go func(a Alerter) {
			a.Alert(ctx, r, lastRecord)
			wg.Done()
		}(a)
	}

	wg.Wait()
}
