package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"text/template"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
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

func SetupProbe(ctx context.Context, tasks []Task) {
	for _, task := range tasks {
		if task.Probe.Target().Scheme == "ping" {
			if err := probe.StartPinger(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "failed to start ping service: %s\n", err)
				os.Exit(1)
			}
			return
		}
	}
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *alertURI != "" {
		alert, err := NewAlert(*alertURI)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid alert target:", err)
			os.Exit(2)
		}
		s.OnIncident = append(s.OnIncident, func(i *api.Incident) {
			go alert.Trigger(ctx, i, s)
		})
	}

	SetupProbe(ctx, tasks)

	if *oneshot {
		os.Exit(RunOneshot(ctx, s, tasks))
	} else {
		os.Exit(RunServer(ctx, s, tasks))
	}
}
