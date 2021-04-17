package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/robfig/cron"
)

var (
	listenPort = flag.Int("p", 9000, "Listen port of status page.")
	storePath  = flag.String("o", "./ayd.log", "Path to log file. Log file is also use for restore status history.")
	oneshot    = flag.Bool("1", false, "Check status only once and exit. Exit with 0 if all check passed, otherwise exit with code 1.")
	alertURI   = flag.String("a", "", "The alert URI that the same format as target URI.")
)

func Usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage:\n")
	fmt.Fprintf(out, "  %s [-p NUMBER | -o FILE]... INTERVALS|TARGETS...\n", os.Args[0])
	fmt.Fprintf(out, "  %s -1 [-o FILE] INTERVALS|TARGETS...\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "OPTIONS:\n")
	flag.PrintDefaults()
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "INTERVALS:\n")
	fmt.Fprintf(out, "  Specify execution schedule in interval (e.g. \"2m\" means \"every 2 minutes\")\n")
	fmt.Fprintf(out, "  or cron expression (e.g. \"*/5 8-19 * * *\" means \"every 5 minutes from 8 p.m. to 7 a.m.\").\n")
	fmt.Fprintf(out, "  Default interval is \"5m\" in if don't pass any interval.\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "TARGETS:\n")
	fmt.Fprintf(out, "  The target address for status checking.\n")
	fmt.Fprintf(out, "  Specify with URI format like \"ping:example.com\" or \"https://example.com/foo/bar\".\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  http, https:\n")
	fmt.Fprintf(out, "   Send HTTP request, and check status code is 2xx or not.\n")
	fmt.Fprintf(out, "   It will follow redirect up to %d times.\n", probe.HTTP_REDIRECT_MAX)
	fmt.Fprintf(out, "   e.g. https://example.com/path/to\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "   You can specify HTTP method in scheme like \"http-head\" or \"https-post\".\n")
	fmt.Fprintf(out, "   Supported method is GET, HEAD, POST, and OPTION. Default is GET method.\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  ping:\n")
	fmt.Fprintf(out, "   Send 4 ICMP echo request in 2 seconds.\n")
	fmt.Fprintf(out, "   e.g. ping:example.com\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  tcp:\n")
	fmt.Fprintf(out, "   Connect to TCP port.\n")
	fmt.Fprintf(out, "   e.g. dns:example.com:3306\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  dns:\n")
	fmt.Fprintf(out, "   Resolve name with DNS.\n")
	fmt.Fprintf(out, "   e.g. dns:example.com\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  exec:\n")
	fmt.Fprintf(out, "   Execute external command.\n")
	fmt.Fprintf(out, "   You can set 1st argument with fragment,\n")
	fmt.Fprintf(out, "   and you can set environment variable with query.\n")
	fmt.Fprintf(out, "   e.g. exec:/path/to/script?something_variable=awesome-value#argument-for-script\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  source:\n")
	fmt.Fprintf(out, "   Load a file, and test target URIs of each lines.\n")
	fmt.Fprintf(out, "   Lines in the file that starts with \"#\" will ignore as comments.\n")
	fmt.Fprintf(out, "   e.g. source:/path/to/list.txt\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "EXAMPLES:\n")
	fmt.Fprintf(out, " Send ping to example.com in default interval(5m):\n")
	fmt.Fprintf(out, "  $ %s ping:example.com\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Send ping to example.com every minutes:\n")
	fmt.Fprintf(out, "  $ %s 1m ping:example.com\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Access to http://example.com every half hours:\n")
	fmt.Fprintf(out, "  $ %s 30m http://example.com\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Check a.local(ping) and b.local(http) every minutes,\n")
	fmt.Fprintf(out, " and execute ./check.sh command every 15 minutes:\n")
	fmt.Fprintf(out, "  $ %s 1m ping:a.local http://b.local 15m exec:./check.sh\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Check targets that listed in file named \"./list.txt\":\n")
	fmt.Fprintf(out, "  $ echo ping:a.local >> list.txt\n")
	fmt.Fprintf(out, "  $ echo ping:b.local >> list.txt\n")
	fmt.Fprintf(out, "  $ %s source:./list.txt\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Listen on http://0.0.0.0:8080 (and connect to example.com:3306 for check):\n")
	fmt.Fprintf(out, "  $ %s -p 8080 1m tcp:example.com:3306\n", os.Args[0])
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

	if *oneshot {
		RunOneshot(s, tasks)
	} else {
		RunServer(s, tasks)
	}
}
