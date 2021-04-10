package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/robfig/cron"
)

var (
	listenPort = flag.Int("p", 9000, "Listen port of status page.")
	storePath  = flag.String("o", "./ayd.log", "Path to log file.")
	oneshot    = flag.Bool("1", false, "Check status only once and exit. Exit with 0 if all check passed, otherwise exit with code 1.")
)

func Usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage: %s [OPTIONS]... INTERVALS|TARGETS...\n", os.Args[0])
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
	fmt.Fprintf(out, "   Send HTTP HEAD request.\n")
	fmt.Fprintf(out, "   e.g. https://example.com/path/to\n")
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
	fmt.Fprintf(out, "   Execute command.\n")
	fmt.Fprintf(out, "   You can set 1st argument with fragment,\n")
	fmt.Fprintf(out, "   and you can set environment variable with query.\n")
	fmt.Fprintf(out, "   e.g. exec:/path/to/script?something_variable=awesome-value#argument-for-script\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "EXAMPLES:\n")
	fmt.Fprintf(out, " Send ping to example.com in default interval(5m):\n")
	fmt.Fprintf(out, "  $ %s example.com\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Send ping to example.com every minutes:\n")
	fmt.Fprintf(out, "  $ %s 1m example.com\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Access to http://example.com every half hours:\n")
	fmt.Fprintf(out, "  $ %s 30m http://example.com\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Check a.local(ping) and b.local(http) every minutes,\n")
	fmt.Fprintf(out, " and check c.local every 15 minutes:\n")
	fmt.Fprintf(out, "  $ %s 1m a.local http://b.local 15m ping:c.local\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, " Listen on http://0.0.0.0:8080 (and connect to example.com:3306 for check):\n")
	fmt.Fprintf(out, "  $ %s -l 8080 1m tcp:example.com:3306\n", os.Args[0])
}

type Task struct {
	Schedule Schedule
	Probe    probe.Probe
}

func ParseArgs(args []string) ([]Task, []error) {
	var result []Task
	var errors []error

	schedule := DEFAULT_SCHEDULE

	for _, a := range args {
		if s, err := ParseSchedule(a); err == nil {
			schedule = s
			continue
		}

		p, err := probe.Get(a)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		result = append(result, Task{
			Schedule: schedule,
			Probe:    p,
		})
	}

	return result, errors
}

func RunOneshot(tasks []Task) {
	var failed atomic.Value
	s := store.New(*storePath)

	wg := &sync.WaitGroup{}
	for _, t := range tasks {
		wg.Add(1)

		f := t.Probe.Check
		go func() {
			r := f()
			s.Append(r)
			if r.Status == store.STATUS_FAIL {
				failed.Store(true)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if failed.Load() != nil {
		os.Exit(1)
	}
}

func RunServer(tasks []Task) {
	listen := fmt.Sprintf("0.0.0.0:%d", *listenPort)
	fmt.Printf("starts Ayd on http://%s\n", listen)

	scheduler := cron.New()
	store := store.New(*storePath)

	for _, t := range tasks {
		fmt.Printf("%s\t%s\n", t.Schedule, t.Probe.Target())

		f := t.Probe.Check
		job := func() {
			store.Append(f())
		}

		if t.Schedule.NeedKickWhenStart() {
			go job()
		}

		scheduler.Schedule(t.Schedule, cron.FuncJob(job))
	}
	fmt.Println()

	if err := store.Restore(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create or open log file: %s\n", err)
		os.Exit(1)
	}

	scheduler.Start()
	defer scheduler.Stop()

	http.ListenAndServe(listen, exporter.New(store))
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	tasks, errors := ParseArgs(flag.Args())
	if errors != nil {
		for _, err := range errors {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(2)
	}
	if len(tasks) == 0 {
		Usage()
		os.Exit(0)
	}

	if *oneshot {
		RunOneshot(tasks)
	} else {
		RunServer(tasks)
	}
}
