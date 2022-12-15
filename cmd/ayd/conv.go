package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/macrat/ayd/internal/logconv"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
)

type ConvCommand struct {
	InStream  io.Reader
	OutStream io.Writer
	ErrStream io.Writer
}

var defaultConvCommand = &ConvCommand{
	InStream:  os.Stdin,
	OutStream: os.Stdout,
	ErrStream: os.Stderr,
}

const ConvHelp = `Ayd conv -- Convert Ayd log file to other format

Usage: ayd conv [OPTIONS...] [INPUT...]

Options:
  -o, --output  Output log file. (default stdout)

  -c, --csv     Convert to CSV. (default format)
  -j, --json    Convert to JSON.
  -l, --ltsv    Convert to LTSV.
  -x, --xlsx    Convert to XLSX.

  -h, --help    Show this help message and exit.
`

func (c ConvCommand) Run(args []string) int {
	flags := pflag.NewFlagSet("ayd conv", pflag.ContinueOnError)

	outputPath := flags.StringP("output", "o", "", "Output log file")

	toCsv := flags.BoolP("csv", "c", false, "Convert to CSV")
	toJson := flags.BoolP("json", "j", false, "Convert to JSON")
	toLtsv := flags.BoolP("ltsv", "l", false, "Convert to LTSV")
	toXlsx := flags.BoolP("xlsx", "x", false, "Convert to XLSX")

	help := flags.BoolP("help", "h", false, "Show this message and exit")

	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(c.ErrStream, err)
		fmt.Fprintf(c.ErrStream, "\nPlease see `%s %s -h` for more information.\n", args[0], args[1])
		return 2
	}

	if *help {
		fmt.Fprint(c.OutStream, ConvHelp)
		return 0
	}

	count := 0
	if *toCsv {
		count++
	}
	if *toJson {
		count++
	}
	if *toLtsv {
		count++
	}
	if *toXlsx {
		count++
	}
	if count > 1 {
		fmt.Fprintln(c.ErrStream, "error: flags for output format can not use multiple in the same time.")
		return 2
	}

	var scanners jointScanner
	for _, path := range flags.Args()[2:] {
		if path == "" || path == "-" {
			scanners = append(scanners, api.NewLogScanner(io.NopCloser(c.InStream)))
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(c.ErrStream, "error: failed to open input log file: %s\n", err)
				return 1
			}
			s := api.NewLogScanner(f)
			scanners = append(scanners, s)
		}
	}
	if len(scanners) == 0 {
		scanners = append(scanners, api.NewLogScanner(io.NopCloser(c.InStream)))
	}
	defer (&scanners).Close()

	output := c.OutStream
	if *outputPath != "" && *outputPath != "-" {
		f, err := os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(c.ErrStream, "error: failed to open output log file: %s\n", err)
			return 1
		}
		defer f.Close()
		output = f
	} else if *toXlsx && isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		fmt.Fprintln(c.ErrStream, "error: can not write xlsx format to stdout. please redirect or use -o option.")
		return 2
	}

	var err error
	switch {
	case *toJson:
		err = c.toJson(&scanners, output)
	case *toLtsv:
		err = c.toLTSV(&scanners, output)
	case *toXlsx:
		err = c.toXlsx(&scanners, output)
	default:
		err = c.toCSV(&scanners, output)
	}
	if err != nil {
		fmt.Fprintf(c.ErrStream, "error: %s\n", err)
		return 1
	} else {
		return 0
	}
}

func (c ConvCommand) toJson(s api.LogScanner, output io.Writer) error {
	if _, err := output.Write([]byte("[\n  ")); err != nil {
		return fmt.Errorf("failed to write log: %s", err)
	}

	first := true

	for s.Scan() {
		if first {
			first = false
		} else {
			if _, err := output.Write([]byte(",\n  ")); err != nil {
				return fmt.Errorf("failed to write log: %s", err)
			}
		}

		if j, err := json.Marshal(s.Record()); err != nil {
			return fmt.Errorf("failed to encode log: %s", err)
		} else if _, err := output.Write(j); err != nil {
			return fmt.Errorf("failed to write log: %s", err)
		}
	}

	if _, err := output.Write([]byte("\n]\n")); err != nil {
		return fmt.Errorf("failed to write log: %s", err)
	}

	return nil
}

func (c ConvCommand) toCSV(s api.LogScanner, output io.Writer) error {
	return logconv.ToCSV(output, s)
}

func (c ConvCommand) toLTSV(s api.LogScanner, output io.Writer) error {
	return logconv.ToLTSV(output, s)
}

func (c ConvCommand) toXlsx(s api.LogScanner, output io.Writer) error {
	return logconv.ToXlsx(output, s, time.Now())
}

type jointScanner []api.LogScanner

func (ss *jointScanner) Scan() bool {
	if len(*ss) == 0 {
		return false
	}

	if (*ss)[0].Scan() {
		return true
	}
	(*ss)[0].Close()
	*ss = (*ss)[1:]

	return ss.Scan()
}

func (ss *jointScanner) Record() api.Record {
	return (*ss)[0].Record()
}

func (ss *jointScanner) Close() error {
	for _, s := range *ss {
		s.Close()
	}
	return nil
}
