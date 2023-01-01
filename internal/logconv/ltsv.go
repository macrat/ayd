package logconv

import (
	"fmt"
	"io"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func ToLTSV(w io.Writer, s api.LogScanner) error {
	for s.Scan() {
		r := s.Record()

		_, err := fmt.Fprintf(
			w,
			"time:%s\tstatus:%s\tlatency:%.3f\ttarget:%s",
			r.Time.Format(time.RFC3339),
			r.Status,
			float64(r.Latency.Microseconds())/1000,
			r.Target,
		)
		if err != nil {
			return err
		}

		if r.Message != "" {
			_, err := fmt.Fprintf(w, "\tmessage:%s", r.Message)
			if err != nil {
				return err
			}
		}

		extra := r.ReadableExtra()

		for _, e := range extra {
			s := e.Value
			if _, ok := r.Extra[e.Key].(string); ok {
				// You should escape if the value is string. Otherwise, it's already escaped as a JSON value.
				s = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(e.Value, `\`, `\\`), "\t", `\t`), "\n", `\n`), "\r", `\r`)
			}
			_, err := fmt.Fprintf(w, "\t%s:%s", e.Key, s)
			if err != nil {
				return err
			}
		}

		_, err = fmt.Fprintln(w)
		if err != nil {
			return err
		}
	}

	return nil
}
