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

func RunServer(ctx context.Context, s *store.Store, tasks []Task, certFile, keyFile string) (exitCode int) {
	protocol := "http"
	if certFile != "" {
		protocol = "https"
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error: certificate file is not exists: %s\n", certFile)
			return 2
		}
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error: key file is not exists: %s\n", keyFile)
			return 2
		}
	}

	ctx, cancel := context.WithCancel(ctx)

	listen := fmt.Sprintf("0.0.0.0:%d", *listenPort)

	scheduler := cron.New()

	if err := s.Restore(); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to read log file: %s\n", err)
		return 1
	}

	fmt.Fprintf(s.Console, "starts Ayd on %s://%s\n", protocol, listen)

	for _, t := range tasks {
		fmt.Fprintf(s.Console, "%s\t%s\n", t.Schedule, t.Probe.Target())

		s.AddTarget(t.Probe.Target())

		job := t.MakeJob(ctx, s)

		if t.Schedule.NeedKickWhenStart() {
			go job.Run()
		}

		scheduler.Schedule(t.Schedule, job)
	}
	fmt.Fprintln(s.Console)

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

	var err error
	if protocol == "https" {
		err = srv.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
	}
	cancel()

	<-cronStopped
	<-httpStopped

	return exitCode
}
