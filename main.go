package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/macrat/ayd/exporter"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

var (
	listenPort = flag.Int("l", 9000, "Listen port of status page.")
	storePath  = flag.String("o", "./ayd.log", "Path to log file.")
)

func Usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage: %s [OPTIONS]... INTERVALS|TARGETS...\n", os.Args[0])
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "OPTIONS:\n")
	flag.PrintDefaults()
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "INTERVALS:\n")
	fmt.Fprintf(out, "  Specify execution interval like \"10m\", \"3h\".\n")
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

func main() {
	flag.Usage = Usage
	flag.Parse()

	scheduler := gocron.NewScheduler(time.UTC)
	store := store.New(*storePath)

	interval := 5 * time.Minute
	errored := false
	for _, x := range flag.Args() {
		if d, err := time.ParseDuration(x); err == nil {
			interval = d
			continue
		}

		t, err := probe.ParseTarget(x)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			errored = true
			continue
		}

		f := probe.Get(t)
		if f == nil {
			fmt.Fprintf(os.Stderr, "unsupported scheme: %s\n", x)
			errored = true
			continue
		}

		scheduler.Every(interval).Do(func() {
			store.Append(f(t))
		})
	}

	if errored {
		os.Exit(1)
	}
	if scheduler.Len() == 0 {
		Usage()
		os.Exit(0)
	}

	fmt.Printf("restore check history from %s...\n", *storePath)
	if err := store.Restore(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create or open log file: %s\n", err)
		os.Exit(2)
	}

	fmt.Printf("start status checking to %d targets...\n", scheduler.Len())
	scheduler.StartAsync()

	listen := fmt.Sprintf("0.0.0.0:%d", *listenPort)
	fmt.Printf("start status page on http://%s...\n", listen)
	http.ListenAndServe(listen, exporter.New(store))
}
