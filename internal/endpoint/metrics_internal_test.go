package endpoint

import (
	"fmt"
	"testing"

	api "github.com/macrat/ayd/lib-ayd"
)

func TestExtractExtraMetrics(t *testing.T) {
	tests := []struct {
		Scheme  string
		Message string
		Want    []extraMetric
	}{
		{
			"http",
			"proto=HTTP/1.1 length=123 status=200_OK",
			[]extraMetric{
				{"http_proto", "gauge", 1.1, ""},
				{"http_content_length_bytes", "gauge", 123, ""},
				{"http_status_code", "gauge", 200, ""},
			},
		},
		{
			"http",
			"proto=HTTP/2.0 length=-1 status=404_Not_Found",
			[]extraMetric{
				{"http_proto", "gauge", 2.0, ""},
				{"http_content_length_bytes", "gauge", -1, ""},
				{"http_status_code", "gauge", 404, ""},
			},
		},
		{
			"https",
			"proto=HTTP/1.0 length=0 status=201_No_Content",
			[]extraMetric{
				{"http_proto", "gauge", 1.0, ""},
				{"http_content_length_bytes", "gauge", 0, ""},
				{"http_status_code", "gauge", 201, ""},
			},
		},
		{
			"https",
			"error: something wrong",
			nil,
		},
		{
			"ftp",
			"type=file size=1234",
			[]extraMetric{
				{"ftp_file_size_bytes", "gauge", 1234, ""},
			},
		},
		{
			"ftp",
			"type=directory files=42",
			[]extraMetric{
				{"ftp_files", "gauge", 42, ""},
			},
		},
		{
			"ftp",
			"type=file size=12",
			[]extraMetric{
				{"ftp_file_size_bytes", "gauge", 12, ""},
			},
		},
		{
			"ftps",
			"oh no",
			nil,
		},
		{
			"ping",
			"ip=127.0.0.1 rtt(min/avg/max)=0.10/0.25/0.50 recv/sent=2/3",
			[]extraMetric{
				{"ping_min_latency_seconds", "gauge", 0.10 / 1000, ""},
				{"ping_average_latency_seconds", "gauge", 0.25 / 1000, ""},
				{"ping_max_latency_seconds", "gauge", 0.50 / 1000, ""},
				{"ping_received_packets", "gauge", 2, ""},
				{"ping_sent_packets", "gauge", 3, ""},
			},
		},
		{
			"ping4",
			"ip=127.0.0.1 rtt(min/avg/max)=1.00/2.00/3.00 recv/sent=9/10",
			[]extraMetric{
				{"ping_min_latency_seconds", "gauge", 1.0 / 1000, ""},
				{"ping_average_latency_seconds", "gauge", 2.0 / 1000, ""},
				{"ping_max_latency_seconds", "gauge", 3.0 / 1000, ""},
				{"ping_received_packets", "gauge", 9, ""},
				{"ping_sent_packets", "gauge", 10, ""},
			},
		},
		{
			"ping6",
			"ip=[::] rtt(min/avg/max)=100.00/200.00/300.00 recv/sent=0/1",
			[]extraMetric{
				{"ping_min_latency_seconds", "gauge", 100.0 / 1000, ""},
				{"ping_average_latency_seconds", "gauge", 200.0 / 1000, ""},
				{"ping_max_latency_seconds", "gauge", 300.0 / 1000, ""},
				{"ping_received_packets", "gauge", 0, ""},
				{"ping_sent_packets", "gauge", 1, ""},
			},
		},
		{
			"ping",
			"failed to send!",
			nil,
		},
		{
			"source",
			"targets=123",
			[]extraMetric{
				{"source_targets", "gauge", 123, ""},
			},
		},
		{
			"source+http",
			"targets=0",
			[]extraMetric{
				{"source_targets", "gauge", 0, ""},
			},
		},
		{
			"source",
			"no such file",
			nil,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d_%s", i, tt.Scheme), func(t *testing.T) {
			actual := extractExtraMetrics(api.Record{
				Target:  &api.URL{Scheme: tt.Scheme},
				Message: tt.Message,
			})

			if len(actual) != len(tt.Want) {
				t.Fatalf("expected %d metrics but got %d metrics", len(tt.Want), len(actual))
			}

			for i := range actual {
				a := actual[i]
				w := tt.Want[i]

				if a.Name != w.Name {
					t.Errorf("%d: expected name is %s but got %s", i, w.Name, a.Name)
				}

				if a.Type != w.Type {
					t.Errorf("%d: expected type is %s but got %s", i, w.Type, a.Type)
				}

				if a.Value != w.Value {
					t.Errorf("%d: expected value is %v but got %v", i, w.Value, a.Value)
				}
			}
		})
	}
}
