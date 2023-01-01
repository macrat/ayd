package main

import (
	"context"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
)

func (cmd *AydCommand) RunOneshot(ctx context.Context, s *store.Store) (exitCode int) {
	var unhealthy atomic.Value

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGHUP)
	defer stop()

	s.OnStatusChanged = append(s.OnStatusChanged, func(r api.Record) {
		if r.Status != api.StatusHealthy {
			unhealthy.Store(true)
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

	if unhealthy.Load() != nil {
		return 1
	} else {
		return 0
	}
}
