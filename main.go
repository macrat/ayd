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

var (
	version = "0.1.0"

	listenPort  = flag.Int("p", 9000, "Listen port of status page.")
	storePath   = flag.String("o", "./ayd.log", "Path to log file. Log file is also use for restore status history.")
	oneshot     = flag.Bool("1", false, "Check status only once and exit. Exit with 0 if all check passed, otherwise exit with code 1.")
	alertURI    = flag.String("a", "", "The alert URI that the same format as target URI.")
	showVersion = flag.Bool("v", false, "Show version and exit.")
)

//go:embed help.txt
var helpText string

func Usage() {
	tmpl := template.Must(template.New("help.txt").Parse(helpText))
	tmpl.Execute(flag.CommandLine.Output(), map[string]interface{}{
		"Command":         os.Args[0],
		"Version":         version,
		"HTTPRedirectMax": probe.HTTP_REDIRECT_MAX,
	})
}

type Task struct {
	Schedule Schedule
	Probe    probe.Probe
}

func (t Task) MakeJob(s *store.Store) cron.Job {
	return cron.FuncJob(func() {
		defer func() {
			if err := recover(); err != nil {
				s.Append(store.Record{
					CheckedAt: time.Now(),
					Target:    t.Probe.Target(),
					Status:    store.STATUS_UNKNOWN,
					Message:   fmt.Sprintf("panic: %s", err),
				})
			}
		}()
		s.Append(t.Probe.Check()...)
	})
}

func (t Task) SameAs(another Task) bool {
	return t.Schedule.String() == another.Schedule.String() && t.Probe.Target().String() == another.Probe.Target().String()
}

func (t Task) In(list []Task) bool {
	for _, x := range list {
		if t.SameAs(x) {
			return true
		}
	}
	return false
}

func ParseArgs(args []string) ([]Task, []error) {
	var tasks []Task
	var errors []error

	schedule := DEFAULT_SCHEDULE

	for _, a := range args {
		if s, err := ParseSchedule(a); err == nil {
			schedule = s
			continue
		}

		p, err := probe.Get(a)
		if err != nil {
			switch err {
			case probe.ErrUnsupportedScheme:
				err = fmt.Errorf("%s: This scheme is not supported.", a)
			case probe.ErrMissingScheme:
				err = fmt.Errorf("%s: Not valid as schedule or target URI. Please specify scheme if this is target. (e.g. ping:example.local or http://example.com)", a)
			case probe.ErrInvalidURI:
				err = fmt.Errorf("%s: Not valid as schedule or target URI.", a)
			default:
				err = fmt.Errorf("%s: %s", a, err)
			}
			errors = append(errors, err)
			continue
		}

		tasks = append(tasks, Task{
			Schedule: schedule,
			Probe:    p,
		})
	}

	var result []Task
	for _, t := range tasks {
		if t.In(result) {
			continue
		}
		result = append(result, t)
	}

	return result, errors
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
		fmt.Println("version:", version)
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
