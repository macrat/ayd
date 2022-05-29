package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/robfig/cron/v3"
)

func (cmd *AydCommand) RunServer(ctx context.Context, s *store.Store) (exitCode int) {
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	listen := fmt.Sprintf("0.0.0.0:%d", cmd.ListenPort)

	scheduler := cron.New()

	if err := s.Restore(); err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: failed to read log file: %s\n", err)
		return 1
	}

	for _, t := range cmd.Tasks {
		fmt.Fprintf(cmd.OutStream, "%s\t%s\n", t.Schedule, t.Prober.Target().String())
	}
	fmt.Fprintln(cmd.OutStream)

	s.Report(&api.URL{Scheme: "ayd", Opaque: "server"}, api.Record{
		Time:    time.Now(),
		Status:  api.StatusHealthy,
		Target:  &api.URL{Scheme: "ayd", Opaque: "server"},
		Message: fmt.Sprintf("Ayd has started"),
		Extra: map[string]interface{}{
			"url": fmt.Sprintf("%s://%s", protocol, listen),
		},
	})

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

		if err := srv.Shutdown(context.Background()); err != nil {
			s.ReportInternalError("endpoint", err.Error())
		}
		wg.Done()
	}()

	var err error
	if protocol == "https" {
		err = srv.ListenAndServeTLS(cmd.CertPath, cmd.KeyPath)
	} else {
		err = srv.ListenAndServe()
	}
	if err != http.ErrServerClosed {
		s.ReportInternalError("endpoint:tls", err.Error())
		exitCode = 1
	}
	cancel()

	wg.Wait()

	return exitCode
}
