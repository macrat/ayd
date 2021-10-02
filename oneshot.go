package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

func RunOneshot(ctx context.Context, s *store.Store, tasks []Task) (exitCode int) {
	var failure atomic.Value
	var unknown atomic.Value

	s.OnIncident = append(s.OnIncident, func(i *api.Incident) {
		switch i.Status {
		case api.StatusFailure:
			failure.Store(true)
		case api.StatusUnknown:
			unknown.Store(true)
		}
	})

	wg := &sync.WaitGroup{}
	for _, t := range tasks {
		wg.Add(1)

		f := t.MakeJob(ctx, s).Run
		go func() {
			f()
			wg.Done()
		}()
	}
	wg.Wait()

	switch {
	case failure.Load() != nil:
		return 1
	case unknown.Load() != nil:
		return 2
	default:
		return 0
	}
}
