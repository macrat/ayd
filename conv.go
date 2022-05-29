package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
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

  -h, --help    Show this help message and exit.
`

func (c ConvCommand) Run(args []string) int {
	flags := pflag.NewFlagSet("ayd conv", pflag.ContinueOnError)

	outputPath := flags.StringP("output", "o", "", "Output log file")

	toCsv := flags.BoolP("csv", "c", false, "Convert to CSV")
	toJson := flags.BoolP("json", "j", false, "Convert to JSON")

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
	if count > 1 {
		fmt.Fprintln(c.ErrStream, "error: flags for output format can not use multiple in the same time.")
		return 2
	}

	var scanners []api.LogScanner
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
			defer s.Close()
			scanners = append(scanners, s)
		}
	}
	if len(scanners) == 0 {
		scanners = append(scanners, api.NewLogScanner(io.NopCloser(c.InStream)))
	}

	output := c.OutStream
	if *outputPath != "" && *outputPath != "-" {
		f, err := os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(c.ErrStream, "error: failed to open output log file: %s\n", err)
			return 1
		}
		defer f.Close()
		output = f
	}

	var err error
	switch {
	case *toJson:
		err = c.toJson(scanners, output)
	default:
		err = c.toCSV(scanners, output)
	}
	if err != nil {
		fmt.Fprintf(c.ErrStream, "error: %s\n", err)
		return 1
	} else {
		return 0
	}
}

func (c ConvCommand) toJson(scanners []api.LogScanner, output io.Writer) error {
	first := true
	for _, s := range scanners {
		for s.Scan() {
			prefix := []byte(",\n  ")
			if first {
				prefix = []byte("[\n  ")
			}
			first = false

			if _, err := output.Write(prefix); err != nil {
				return fmt.Errorf("failed to write log: %s", err)
			}

			if j, err := json.Marshal(s.Record()); err != nil {
				return fmt.Errorf("failed to encode log: %s", err)
			} else if _, err := output.Write(j); err != nil {
				return fmt.Errorf("failed to write log: %s", err)
			}
		}
	}

	if _, err := output.Write([]byte("\n]\n")); err != nil {
		return fmt.Errorf("failed to write log: %s", err)
	}

	return nil
}

func (c ConvCommand) toCSV(scanners []api.LogScanner, output io.Writer) error {
	writer := csv.NewWriter(output)

	for _, s := range scanners {
		for s.Scan() {
			r := s.Record()
			err := writer.Write([]string{
				r.CheckedAt.Format(time.RFC3339),
				r.Status.String(),
				strconv.FormatFloat(float64(r.Latency.Microseconds())/1000, 'f', 3, 64),
				r.Target.String(),
				r.ReadableMessage(),
			})
			if err != nil {
				return fmt.Errorf("failed to write log: %s", err)
			}
		}
	}

	writer.Flush()

	return nil
}
