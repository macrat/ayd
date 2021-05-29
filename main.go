package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"text/template"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
)

var (
	version = "HEAD"
	commit  = "UNKNOWN"

	listenPort  = flag.Int("p", 9000, "Listen port of status page.")
	storePath   = flag.String("o", "./ayd.log", "Path to log file. Log file is also use for restore status history. Ayd won't create log file if set \"-\" or empty.")
	oneshot     = flag.Bool("1", false, "Check status only once and exit. Exit with 0 if all check passed, otherwise exit with code 1.")
	alertURL    = flag.String("a", "", "The alert URL that the same format as target URL.")
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
			if err := probe.CheckPingPermission(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to start ping service: %s\n", err)
				os.Exit(1)
			}
			return
		}
	}
}

func RunAyd() int {
	flag.Usage = Usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("Ayd? version %s (%s)\n", version, commit)
		return 0
	}

	if *oneshot && *alertURL != "" {
		fmt.Fprintln(os.Stderr, "warning: -a option will ignored when use with -1 option")
	}
	if *oneshot && *listenPort != 9000 {
		fmt.Fprintln(os.Stderr, "warning: -p option will ignored when use with -1 option")
	}

	tasks, errors := ParseArgs(flag.Args())
	if errors != nil {
		fmt.Fprintln(os.Stderr, "Invalid argument:")
		for _, err := range errors {
			fmt.Fprintln(os.Stderr, "", err)
		}
		fmt.Fprintf(os.Stderr, "\nPlease see `%s -h` for more information.\n", os.Args[0])
		return 2
	}
	if len(tasks) == 0 {
		Usage()
		return 0
	}

	if *storePath == "-" {
		*storePath = ""
	}
	s, err := store.New(*storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %s\n", err)
		return 1
	}
	defer s.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *alertURL != "" {
		alert, err := NewAlert(*alertURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid alert target:", err)
			return 2
		}
		s.OnIncident = append(s.OnIncident, func(i *api.Incident) {
			go alert.Trigger(ctx, i, s)
		})
	}

	SetupProbe(ctx, tasks)

	if *oneshot {
		return RunOneshot(ctx, s, tasks)
	} else {
		return RunServer(ctx, s, tasks)
	}
}

func main() {
	os.Exit(RunAyd())
}
