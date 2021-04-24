package main

import (
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/robfig/cron"
)

const (
	TASK_TIMEOUT = 1 * time.Hour
)

var (
	version = "HEAD"
	commit  = "UNKNOWN"

	listenPort  = flag.Int("p", 9000, "Listen port of status page.")
	storePath   = flag.String("o", "./ayd.log", "Path to log file. Log file is also use for restore status history.")
	oneshot     = flag.Bool("1", false, "Check status only once and exit. Exit with 0 if all check passed, otherwise exit with code 1.")
	alertURI    = flag.String("a", "", "The alert URI that the same format as target URI.")
	showVersion = flag.Bool("v", false, "Show version and exit.")
)

//go:embed help.txt
var helpText string

func init() {
	probe.HTTPUserAgent = fmt.Sprintf("ayd/%s health check", version)
}

func Usage() {
	tmpl := template.Must(template.New("help.txt").Parse(helpText))
	tmpl.Execute(flag.CommandLine.Output(), map[string]interface{}{
		"Command":         os.Args[0],
		"Version":         version,
		"HTTPRedirectMax": probe.HTTP_REDIRECT_MAX,
	})
}

func RunOneshot(s *store.Store, tasks []Task) {
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

	if failure.Load() != nil {
		os.Exit(1)
	}
	if unknown.Load() != nil {
		os.Exit(2)
	}
}

func RunServer(s *store.Store, tasks []Task) {
	listen := fmt.Sprintf("0.0.0.0:%d", *listenPort)
	fmt.Printf("starts Ayd on http://%s\n", listen)

	scheduler := cron.New()

	if err := s.Restore(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read log file: %s\n", err)
		os.Exit(1)
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
	os.Exit(1)
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("Ayd? version %s (%s)\n", version, commit)
		os.Exit(0)
	}

	tasks, errors := ParseArgs(flag.Args())
	if errors != nil {
		fmt.Fprintln(os.Stderr, "Invalid argument:")
		for _, err := range errors {
			fmt.Fprintln(os.Stderr, "", err)
		}
		fmt.Fprintf(os.Stderr, "\nPlease see `%s -h` for more information.\n", os.Args[0])
		os.Exit(2)
	}
	if len(tasks) == 0 {
		Usage()
		os.Exit(0)
	}

	s, err := store.New(*storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %s\n", err)
		os.Exit(1)
	}
	defer s.Close()

	if *alertURI != "" {
		alert, err := NewAlert(*alertURI)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid alert target:", err)
			os.Exit(2)
		}
		s.OnIncident = append(s.OnIncident, alert.Trigger)
	}

	if *oneshot {
		RunOneshot(s, tasks)
	} else {
		RunServer(s, tasks)
	}
}
