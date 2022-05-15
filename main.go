package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/signal"
	"text/template"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/store"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/spf13/pflag"
)

var (
	version = "HEAD"
	commit  = "UNKNOWN"
)

func init() {
	scheme.HTTPUserAgent = fmt.Sprintf("ayd/%s health check", version)
}

type AydCommand struct {
	OutStream io.Writer
	ErrStream io.Writer

	ListenPort  int
	StorePath   string
	OneshotMode bool
	AlertURLs   []string
	UserInfo    string
	CertPath    string
	KeyPath     string
	ShowVersion bool
	ShowHelp    bool

	Tasks []Task
}

var defaultAydCommand = &AydCommand{
	OutStream: os.Stdout,
	ErrStream: os.Stderr,
}

//go:embed help.txt
var helpText string

func (cmd *AydCommand) PrintUsage(detail bool) {
	tmpl := template.Must(template.New("help.txt").Parse(helpText))
	tmpl.Execute(cmd.ErrStream, map[string]interface{}{
		"Version":         version,
		"HTTPRedirectMax": scheme.HTTP_REDIRECT_MAX,
		"Short":           !detail,
	})
}

func (cmd *AydCommand) ParseArgs(args []string) (exitCode int) {
	flags := pflag.NewFlagSet("ayd", pflag.ContinueOnError)

	flags.IntVarP(&cmd.ListenPort, "port", "p", 9000, "HTTP listen port")
	flags.StringVarP(&cmd.StorePath, "log-file", "f", "./ayd.log", "Path to log file")
	flags.BoolVarP(&cmd.OneshotMode, "oneshot", "1", false, "Check status only once and exit")
	flags.StringArrayVarP(&cmd.AlertURLs, "alert", "a", nil, "The alert URLs")
	flags.StringVarP(&cmd.UserInfo, "user", "u", "", "Username and password for HTTP endpoint")
	flags.StringVarP(&cmd.CertPath, "ssl-cert", "c", "", "HTTPS certificate file")
	flags.StringVarP(&cmd.KeyPath, "ssl-key", "k", "", "HTTPS key file")
	flags.BoolVarP(&cmd.ShowVersion, "version", "v", false, "Show version")
	flags.BoolVarP(&cmd.ShowHelp, "help", "h", false, "Show help message")

	// TODO: remove -o option before to release version 1.0.0
	compatPath := flags.StringP("deprecated-output-file", "o", "", "DEPRECATED: This option is renamed to -f.")

	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintln(cmd.ErrStream, err)
		fmt.Fprintf(cmd.ErrStream, "\nPlease see `%s -h` for more information.\n", args[0])
		return 2
	}

	if cmd.ShowVersion || cmd.ShowHelp {
		return 0
	}

	if cmd.OneshotMode {
		if flags.Changed("port") {
			fmt.Fprintln(cmd.ErrStream, "warning: port option will ignored in the oneshot mode.")
		}
		if flags.Changed("user") {
			fmt.Fprintln(cmd.ErrStream, "warning: user option will ignored in the oneshot mode.")
		}
		if flags.Changed("ssl-cert") || flags.Changed("ssl-key") {
			fmt.Fprintln(cmd.ErrStream, "warning: ssl cert and key options will ignored in the oneshot mode.")
		}
	} else {
		if cmd.CertPath != "" && cmd.KeyPath == "" || cmd.CertPath == "" && cmd.KeyPath != "" {
			fmt.Fprintln(cmd.ErrStream, "invalid argument: the both of -c and -k option is required if you want to use HTTPS.")
			return 2
		}
	}

	if flags.Changed("deprecated-output-file") {
		fmt.Fprintf(cmd.ErrStream, "\nwarning: The -o option is deprecated.\n         Please use -f option instead of -o.\n\n")
	}
	if !flags.Changed("log-file") && *compatPath != "" {
		cmd.StorePath = *compatPath
	}
	if cmd.StorePath == "-" {
		cmd.StorePath = ""
	}

	var err error
	cmd.Tasks, err = ParseArgs(flags.Args())
	if err != nil {
		fmt.Fprintln(cmd.ErrStream, err.Error())
		fmt.Fprintf(cmd.ErrStream, "\nPlease see `%s -h` for more information.\n", args[0])
		return 2
	}
	if len(cmd.Tasks) == 0 {
		cmd.PrintUsage(false)
		return 2
	}

	return 0
}

func (cmd *AydCommand) PrintVersion() {
	fmt.Fprintf(cmd.OutStream, "Ayd version %s (%s)\n", version, commit)
}

func (cmd *AydCommand) Run(args []string) (exitCode int) {
	if code := cmd.ParseArgs(args); code != 0 {
		return code
	}

	if cmd.ShowVersion {
		cmd.PrintVersion()
		return 0
	}

	if cmd.ShowHelp {
		cmd.PrintUsage(true)
		return 0
	}

	s, err := store.New(cmd.StorePath, cmd.OutStream)
	if err != nil {
		fmt.Fprintf(cmd.ErrStream, "error: failed to open log file: %s\n", err)
		return 1
	}
	defer s.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if len(cmd.AlertURLs) > 0 {
		alert, err := scheme.NewAlerterSet(cmd.AlertURLs)
		if err != nil {
			fmt.Fprintln(cmd.ErrStream, err)
			return 2
		}
		s.OnStatusChanged = append(s.OnStatusChanged, func(r api.Record) {
			alert.Alert(ctx, s, r)
		})
	}

	if cmd.OneshotMode {
		return cmd.RunOneshot(ctx, s)
	} else {
		return cmd.RunServer(ctx, s)
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "oneshot":
			os.Args[1] = "-1"
			os.Exit(defaultAydCommand.Run(os.Args))
		case "conv":
			os.Exit(defaultConvCommand.Run(os.Args))
		}
	}

	os.Exit(defaultAydCommand.Run(os.Args))
}
