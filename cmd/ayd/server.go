package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/meta"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/robfig/cron/v3"
)

func (cmd *AydCommand) reportStartServer(s *store.Store, protocol, listen string) {
	tasks := make(map[string][]string)

	for _, t := range cmd.Tasks {
		k := t.Schedule.String()
		if l, ok := tasks[k]; ok {
			tasks[k] = append(l, t.Prober.Target().String())
		} else {
			tasks[k] = []string{t.Prober.Target().String()}
		}
	}

	cmd.StartedAt = time.Now()

	u := &api.URL{Scheme: "ayd", Opaque: "server"}
	s.Report(u, api.Record{
		Time:    cmd.StartedAt,
		Status:  api.StatusHealthy,
		Target:  u,
		Message: "start Ayd server",
		Extra: map[string]interface{}{
			"url":     fmt.Sprintf("%s://%s", protocol, listen),
			"targets": tasks,
			"version": fmt.Sprintf("%s (%s)", meta.Version, meta.Commit),
		},
	})
}

func (cmd *AydCommand) reportStopServer(s *store.Store, protocol, listen string) {
	u := &api.URL{Scheme: "ayd", Opaque: "server"}
	s.Report(u, api.Record{
		Time:    time.Now(),
		Status:  api.StatusHealthy,
		Target:  u,
		Message: "stop Ayd server",
		Extra: map[string]interface{}{
			"url":     fmt.Sprintf("%s://%s", protocol, listen),
			"version": fmt.Sprintf("%s (%s)", meta.Version, meta.Commit),
			"since":   cmd.StartedAt.Format(time.RFC3339),
		},
	})
}

func (cmd *AydCommand) RunServer(ctx context.Context, s *store.Store) (exitCode int) {
	startDebugLogger(s)

	protocol := "http"
	if cmd.CertPath != "" {
		protocol = "https"
		if _, err := os.Stat(cmd.CertPath); os.IsNotExist(err) {
			fmt.Fprintf(cmd.ErrStream, "error: certificate file does not exist: %s\n", cmd.CertPath)
			return 2
		}
		if _, err := os.Stat(cmd.KeyPath); os.IsNotExist(err) {
			fmt.Fprintf(cmd.ErrStream, "error: key file does not exist: %s\n", cmd.KeyPath)
			return 2
		}
	}

	scheduler := cron.New()

	if err := s.Restore(); err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: failed to read log file: %s\n", err)
		return 1
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cmd.ListenPort))
	if err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: failed to start HTTP server: %s\n", err)
		return 2
	}
	listen := listener.Addr().String()

	cmd.reportStartServer(s, protocol, listen)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGHUP)
		select {
		case <-ctx.Done():
		case <-ch:
			cmd.reportStopServer(s, protocol, listen)
			cancel()
		}
	}()
	defer cancel()

	wg := &sync.WaitGroup{}
	for _, t := range cmd.Tasks {
		s.ActivateTarget(t.Prober.Target(), t.Prober.Target())

		job := t.MakeJob(ctx, s)

		if t.Schedule.NeedKickWhenStart() {
			wg.Add(1)
			go func() {
				job.Run()
				wg.Done()
			}()
		}

		scheduler.Schedule(t.Schedule, job)
	}

	scheduler.Start()
	defer scheduler.Stop()

	srv := &http.Server{Addr: listen, Handler: endpoint.WithBasicAuth(endpoint.New(s), cmd.UserInfo)}

	wg.Add(2)
	go func() {
		<-ctx.Done()

		go func() {
			<-scheduler.Stop().Done()
			wg.Done()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			s.ReportInternalError("endpoint", fmt.Sprintf("failed to graceful shutdown: %s", err.Error()))
		}
		wg.Done()
	}()

	if protocol == "https" {
		err = srv.ServeTLS(listener, cmd.CertPath, cmd.KeyPath)
	} else {
		err = srv.Serve(listener)
	}
	if err != http.ErrServerClosed {
		s.ReportInternalError("endpoint", err.Error())
		exitCode = 1
	}
	cancel()

	wg.Wait()

	return exitCode
}
