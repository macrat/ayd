package logconv

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

func ToCSV(w io.Writer, s api.LogScanner) error {
	c := csv.NewWriter(w)

	err := c.Write([]string{"time", "status", "latency", "target", "message", "extra"})
	if err != nil {
		return err
	}

	for s.Scan() {
		r := s.Record()

		var extra []byte
		if len(r.Extra) > 0 {
			// Ignore error because it use empty string if failed to convert.
			extra, _ = json.Marshal(r.Extra)
		}

		err := c.Write([]string{
			r.Time.Format(time.RFC3339),
			r.Status.String(),
			strconv.FormatFloat(float64(r.Latency.Microseconds())/1000, 'f', 3, 64),
			r.Target.String(),
			r.Message,
			string(extra),
		})
		if err != nil {
			return err
		}
	}

	c.Flush()

	return nil
}
