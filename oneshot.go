package main

import (
	"sync"
	"sync/atomic"

	"github.com/macrat/ayd/store"
)

func RunOneshot(s *store.Store, tasks []Task) (exitCode int) {
	var failure atomic.Value
	var unknown atomic.Value

	s.OnIncident = append(s.OnIncident, func(i *store.Incident) []store.Record {
		switch i.Status {
		case store.STATUS_FAILURE:
			failure.Store(true)
		case store.STATUS_UNKNOWN:
			unknown.Store(true)
		}
		return nil
	})

	wg := &sync.WaitGroup{}
	for _, t := range tasks {
		wg.Add(1)

		f := t.MakeJob(s).Run
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
