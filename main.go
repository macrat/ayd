package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"text/template"

	"github.com/macrat/ayd/internal/alert"
	"github.com/macrat/ayd/internal/probe"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/spf13/pflag"
)

var (
	version = "HEAD"
	commit  = "UNKNOWN"

	listenPort  = pflag.IntP("port", "p", 9000, "HTTP listen port")
	storePath   = pflag.StringP("log-file", "f", "./ayd.log", "Path to log file")
	oneshot     = pflag.BoolP("oneshot", "1", false, "Check status only once and exit")
	alertURL    = pflag.StringP("alert", "a", "", "The alert URL")
	userinfo    = pflag.StringP("user", "u", "", "Username and password for HTTP endpoint")
	certPath    = pflag.StringP("ssl-cert", "c", "", "HTTPS certificate file")
	keyPath     = pflag.StringP("ssl-key", "k", "", "HTTPS key file")
	showVersion = pflag.BoolP("verbose", "v", false, "Show version")
	showHelp    = pflag.BoolP("help", "h", false, "Show help message")

	// TODO: remove -o option before to release version 1.0.0
	compatPath = pflag.StringP("deprecated-output-file", "o", "", "DEPRECATED: This option is renamed to -f.")
)

//go:embed help.txt
var helpText string

func init() {
	probe.HTTPUserAgent = fmt.Sprintf("ayd/%s health check", version)
}

func Usage() {
	tmpl := template.Must(template.New("help.txt").Parse(helpText))
	tmpl.Execute(os.Stderr, map[string]interface{}{
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
	pflag.Usage = Usage
	pflag.Parse()

	if *showHelp {
		Usage()
		return 0
	}

	if *showVersion {
		fmt.Printf("Ayd? version %s (%s)\n", version, commit)
		return 0
	}

	if *oneshot {
		if *listenPort != 9000 {
			fmt.Fprintln(os.Stderr, "warning: port option will ignored when use with -1 option")
		}
		if *userinfo != "" {
			fmt.Fprintln(os.Stderr, "warning: user option will ignored in the oneshot mode")
		}
		if *certPath != "" || *keyPath != "" {
			fmt.Fprintln(os.Stderr, "warning: ssl cert and key options will ignored in the oneshot mode")
		}
	} else {
		if *certPath != "" && *keyPath == "" || *certPath == "" && *keyPath != "" {
			fmt.Fprintln(os.Stderr, "Invalid argument:")
			fmt.Fprintln(os.Stderr, " You can't use only -k option or only -c option. Please set both of them if you want to use HTTPS.")
			return 2
		}
	}

	tasks, err := ParseArgs(pflag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stderr, "\nPlease see `%s -h` for more information.\n", os.Args[0])
		return 2
	}
	if len(tasks) == 0 {
		Usage()
		return 0
	}

	if *storePath == "./ayd.log" && *compatPath != "" {
		fmt.Fprintf(os.Stderr, "\nwarning: The -o option is deprecated.\n         Please use -f option instead of -o.\n\n")
		*storePath = *compatPath
	}
	if *storePath == "-" {
		*storePath = ""
	}
	s, err := store.New(*storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open log file: %s\n", err)
		return 1
	}
	defer s.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *alertURL != "" {
		alert, err := alert.New(*alertURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid alert target:", err)
			return 2
		}
		s.OnIncident = append(s.OnIncident, func(i *api.Incident) {
			alert.Trigger(ctx, i, s)
		})
	}

	SetupProbe(ctx, tasks)

	if *oneshot {
		return RunOneshot(ctx, s, tasks)
	} else {
		return RunServer(ctx, s, tasks, *certPath, *keyPath)
	}
}

func main() {
	os.Exit(RunAyd())
}
