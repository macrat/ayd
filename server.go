package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/store"
	"github.com/robfig/cron/v3"
)

func RunServer(ctx context.Context, s *store.Store, tasks []Task) (exitCode int) {
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

		job := t.MakeJob(ctx, s)

		if t.Schedule.NeedKickWhenStart() {
			go job.Run()
		}

		scheduler.Schedule(t.Schedule, job)
	}
	fmt.Println()

	cronStopped := make(chan struct{})
	httpStopped := make(chan struct{})

	scheduler.Start()
	defer scheduler.Stop()
	go func() {
		<-ctx.Done()
		<-scheduler.Stop().Done()
		close(cronStopped)
	}()

	srv := &http.Server{Addr: listen, Handler: exporter.New(s)}
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		close(httpStopped)
	}()
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
	}

	<-cronStopped
	<-httpStopped

	return exitCode
}
