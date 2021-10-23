package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

func (cmd *AydCommand) RunOneshot(ctx context.Context, s *store.Store) (exitCode int) {
	var failure atomic.Value
	var unknown atomic.Value

	s.OnStatusChanged = append(s.OnStatusChanged, func(r api.Record) {
		switch r.Status {
		case api.StatusFailure:
			failure.Store(true)
		case api.StatusUnknown:
			unknown.Store(true)
		}
	})

	wg := &sync.WaitGroup{}
	for _, t := range cmd.Tasks {
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
