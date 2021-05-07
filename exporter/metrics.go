package exporter

import (
	"fmt"
	"net/http"
	"strings"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/store"
)

func MetricsExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, hs := range s.ProbeHistory() {
			if len(hs.Records) > 0 {
				last := hs.Records[len(hs.Records)-1]

				up := 0
				if last.Status == api.StatusHealthy {
					up = 1
				}
				latency := last.Latency.Seconds()
				target := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(hs.Target.String(), "\\", "\\\\"), "\n", "\\\n"), "\"", "\\\"")

				timestamp := last.CheckedAt.Unix()

				fmt.Fprintln(w, "# HELP ayd_healthy 1 if target is healthy, otherwise 0.")
				fmt.Fprintln(w, "# TYPE ayd_healthy Gauge")
				fmt.Fprintf(w, "ayd_healthy{target=\"%s\"} %d %d\n", target, up, timestamp)
				fmt.Fprintln(w, "# HELP ayd_latency_seconds A duration in seconds that taken checking for the target.")
				fmt.Fprintln(w, "# TYPE ayd_latency_seconds Gauge")
				fmt.Fprintf(w, "ayd_latency_seconds{target=\"%s\"} %f %d\n", hs.Target, latency, timestamp)
				fmt.Fprintln(w)
			}
		}

		fmt.Fprintln(w, "# HELP ayd_healthy 1 if target is healthy, otherwise 0.")
		fmt.Fprintln(w, "# TYPE ayd_healthy Gauge")
		if s.Err() == nil {
			fmt.Fprintln(w, `ayd_healthy{target="ayd"} 1`)
		} else {
			fmt.Fprintln(w, `ayd_healthy{target="ayd"} 0`)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "# HELP ayd_incident_count The number of incident happend since server started")
		fmt.Fprintln(w, "# TYPE ayd_incident_count Counter")
		fmt.Fprintf(w, "ayd_incident_count %d\n", s.IncidentCount)
	}
}
