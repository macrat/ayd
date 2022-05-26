package endpoint

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	api "github.com/macrat/ayd/lib-ayd"
)

type extraMetric struct {
	Name  string
	Type  string
	Value float64
	Help  string
}

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
	Extra     []extraMetric
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
					Timestamp: last.CheckedAt.UnixMilli(),
					Latency:   last.Latency.Seconds(),
					Target:    strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(hs.Target.String(), "\\", "\\\\"), "\n", "\\\n"), "\"", "\\\""),
					Extra:     extractExtraMetrics(last),
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

		for _, m := range metrics {
			fmt.Fprintln(w)
			for _, e := range m.Extra {
				fmt.Fprintf(w, "# HELP ayd_%s %s\n", e.Name, e.Help)
				fmt.Fprintf(w, "# TYPE ayd_%s %s\n", e.Name, e.Type)
				fmt.Fprintf(w, "ayd_%s{target=\"%s\"} %v %d\n", e.Name, m.Target, e.Value, m.Timestamp)
			}
		}
	}
}

var (
	httpMessageRe   = regexp.MustCompile(`^proto=HTTP/([0-9]+\.[0-9]+) length=(-?[0-9]+) status=([0-9]+)_`)
	ftpMessageRe    = regexp.MustCompile(`^type=(file|directory) (?:size|files)=([0-9]+)$`)
	pingMessageRe   = regexp.MustCompile(`^ip=[^ ]+ rtt\(min/avg/max\)=([0-9]+\.[0-9]{2})/([0-9]+\.[0-9]{2})/([0-9]+\.[0-9]{2}) recv/sent=([0-9]+)/([0-9]+)$`)
	sourceMessageRe = regexp.MustCompile(`^targets=([0-9]+)$`)
)

func parseFloats(ss []string) ([]float64, error) {
	buf := make([]float64, len(ss))
	for i := range ss {
		var err error
		buf[i], err = strconv.ParseFloat(ss[i], 64)
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}

func extractExtraMetrics(r api.Record) []extraMetric {
	switch r.Target.Scheme {
	case "http", "https":
		m := httpMessageRe.FindStringSubmatch(r.Message)
		if len(m) > 0 {
			if f, err := parseFloats(m[1:]); err == nil {
				return []extraMetric{
					{"http_proto", "gauge", f[0], "HTTP protocol version."},
					{"http_content_length_bytes", "gauge", f[1], "HTTP Content-Length in the response header."},
					{"http_status_code", "gauge", f[2], "The response status code."},
				}
			}
		}
	case "ftp", "ftps":
		m := ftpMessageRe.FindStringSubmatch(r.Message)
		if len(m) > 0 {
			if f, err := strconv.ParseFloat(m[2], 64); err == nil {
				if m[1] == "directory" {
					return []extraMetric{
						{"ftp_files", "gauge", f, "The number of files in the target directory."},
					}
				} else {
					return []extraMetric{
						{"ftp_file_size_bytes", "gauge", f, "The size of the target file."},
					}
				}
			}
		}
	case "ping", "ping4", "ping6":
		m := pingMessageRe.FindStringSubmatch(r.Message)
		if len(m) > 0 {
			if f, err := parseFloats(m[1:]); err == nil {
				return []extraMetric{
					{"ping_min_latency_seconds", "gauge", f[0] / 1000.0, "The minimal latency in seconds."},
					{"ping_average_latency_seconds", "gauge", f[1] / 1000.0, "The average latency in seconds."},
					{"ping_max_latency_seconds", "gauge", f[2] / 1000.0, "The maximum latency in seconds."},
					{"ping_received_packets", "gauge", f[3], "Number of packets that received in the latest probe."},
					{"ping_sent_packets", "gauge", f[4], "Number of packets that sent in the latest probe."},
				}
			}
		}
	case "source", "source+http", "source+https", "source+ftp", "source+ftps", "source+exec":
		m := sourceMessageRe.FindStringSubmatch(r.Message)
		if len(m) > 0 {
			if f, err := strconv.ParseFloat(m[1], 64); err == nil {
				return []extraMetric{
					{"source_targets", "gauge", f, "The number of loaded targets."},
				}
			}
		}
	}
	return nil
}
