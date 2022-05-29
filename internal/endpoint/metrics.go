package endpoint

import (
	"fmt"
	"net/http"
	"strings"

	api "github.com/macrat/ayd/lib-ayd"
)

// metricInfo is a metric point for /metrics endpoint.
type metricInfo struct {
	Timestamp int64
	Target    string
	Healthy   int
	Unknown   int
	Degrade   int
	Failure   int
	Aborted   int
	Latency   float64
}

// MetricsEndpoint implements Prometheus metrics endpoint.
// This endpoint follows both of Prometheus specification and OpenMetrics specification.
func MetricsEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hss := s.ProbeHistory()
		metrics := make([]metricInfo, 0, len(hss))
		for _, hs := range hss {
			if len(hs.Records) > 0 {
				last := hs.Records[len(hs.Records)-1]

				m := metricInfo{
					Timestamp: last.Time.UnixMilli(),
					Latency:   last.Latency.Seconds(),
					Target:    strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(hs.Target.String(), "\\", "\\\\"), "\n", "\\\n"), "\"", "\\\""),
				}

				switch last.Status {
				case api.StatusHealthy:
					m.Healthy = 1
				case api.StatusUnknown:
					m.Unknown = 1
				case api.StatusDegrade:
					m.Degrade = 1
				case api.StatusFailure:
					m.Failure = 1
				case api.StatusAborted:
					m.Aborted = 1
				}

				metrics = append(metrics, m)
			}
		}

		fmt.Fprintln(w, "# HELP ayd_status The target status.")
		fmt.Fprintln(w, "# TYPE ayd_status gauge")
		for _, m := range metrics {
			fmt.Fprintf(w, "ayd_status{target=\"%s\",status=\"healthy\"} %d %d\n", m.Target, m.Healthy, m.Timestamp)
			fmt.Fprintf(w, "ayd_status{target=\"%s\",status=\"unknown\"} %d %d\n", m.Target, m.Unknown, m.Timestamp)
			fmt.Fprintf(w, "ayd_status{target=\"%s\",status=\"degrade\"} %d %d\n", m.Target, m.Degrade, m.Timestamp)
			fmt.Fprintf(w, "ayd_status{target=\"%s\",status=\"failure\"} %d %d\n", m.Target, m.Failure, m.Timestamp)
			fmt.Fprintf(w, "ayd_status{target=\"%s\",status=\"aborted\"} %d %d\n", m.Target, m.Aborted, m.Timestamp)
		}
		if healthy, _ := s.Errors(); healthy {
			fmt.Fprintln(w, `ayd_status{target="ayd",status="healthy"} 1`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="unknown"} 0`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="degrade"} 0`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="failure"} 0`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="aborted"} 0`)
		} else {
			fmt.Fprintln(w, `ayd_status{target="ayd",status="healthy"} 0`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="unknown"} 0`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="degrade"} 0`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="failure"} 1`)
			fmt.Fprintln(w, `ayd_status{target="ayd",status="aborted"} 0`)
		}
		fmt.Fprintln(w)

		fmt.Fprintln(w, "# HELP ayd_latency_seconds The duration in seconds that taken checking for the target.")
		fmt.Fprintln(w, "# TYPE ayd_latency_seconds gauge")
		fmt.Fprintln(w, "# UNIT ayd_latency_seconds seconds")
		for _, m := range metrics {
			fmt.Fprintf(w, "ayd_latency_seconds{target=\"%s\"} %f %d\n", m.Target, m.Latency, m.Timestamp)
		}
		fmt.Fprintln(w)

		fmt.Fprintln(w, "# HELP ayd_incident_total The number of incident happened since server started.")
		fmt.Fprintln(w, "# TYPE ayd_incident_total counter")
		fmt.Fprintf(w, "ayd_incident_total %d\n", s.IncidentCount())
	}
}
