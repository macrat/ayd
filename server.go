package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/store"
	"github.com/robfig/cron/v3"
)

func RunServer(s *store.Store, tasks []Task) (exitCode int) {
	listen := fmt.Sprintf("0.0.0.0:%d", *listenPort)
	fmt.Printf("starts Ayd on http://%s\n", listen)

	scheduler := cron.New()

	if err := s.Restore(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read log file: %s\n", err)
		return 1
	}

	for _, t := range tasks {
		fmt.Printf("%s\t%s\n", t.Schedule, t.Probe.Target())

		s.AddTarget(t.Probe.Target())

		job := t.MakeJob(s)

		if t.Schedule.NeedKickWhenStart() {
			go job.Run()
		}

		scheduler.Schedule(t.Schedule, job)
	}
	fmt.Println()

	scheduler.Start()
	defer scheduler.Stop()

	fmt.Fprintln(os.Stderr, http.ListenAndServe(listen, exporter.New(s)))
	return 1
}
