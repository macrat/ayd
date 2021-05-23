package ayd

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/macrat/ayd/store/freeze"
)

func TestConvertIncident(t *testing.T) {
	tests := []struct {
		Name   string
		Input  freeze.Incident
		Output Incident
		Error  string
	}{
		{
			"not-yet-resolved",
			freeze.Incident{
				Target:   "foo:bar",
				Status:   "FAILURE",
				Message:  "this is failure",
				CausedAt: "2001-02-03T00:02:00Z",
			},
			Incident{
				Target:   &url.URL{Scheme: "foo", Opaque: "bar"},
				Status:   StatusFailure,
				Message:  "this is failure",
				CausedAt: time.Date(2001, 2, 3, 0, 2, 0, 0, time.UTC),
			},
			"",
		},
		{
			"already-resolved",
			freeze.Incident{
				Target:     "foo:bar",
				Status:     "FAILURE",
				Message:    "this is also failure",
				CausedAt:   "2001-02-03T00:01:00Z",
				ResolvedAt: "2001-02-03T00:02:00Z",
			},
			Incident{
				Target:     &url.URL{Scheme: "foo", Opaque: "bar"},
				Status:     StatusFailure,
				Message:    "this is also failure",
				CausedAt:   time.Date(2001, 2, 3, 0, 1, 0, 0, time.UTC),
				ResolvedAt: time.Date(2001, 2, 3, 0, 2, 0, 0, time.UTC),
			},
			"",
		},
		{
			"invalid-target",
			freeze.Incident{
				Target:     "::",
				Status:     "UNKNOWN",
				Message:    "this is invalid URL",
				CausedAt:   "2001-02-03T00:01:00Z",
				ResolvedAt: "2001-02-03T00:02:00Z",
			},
			Incident{},
			`invalid target URL: parse "::": missing protocol scheme`,
		},
		{
			"invalid-caused-at",
			freeze.Incident{
				Target:     "foo:bar",
				Status:     "FAILURE",
				Message:    "this is also failure",
				CausedAt:   "this is not a time",
				ResolvedAt: "2001-02-03T00:02:00Z",
			},
			Incident{},
			`caused time is invalid: parsing time "this is not a time" as "2006-01-02T15:04:05Z07:00": cannot parse "this is not a time" as "2006"`,
		},
		{
			"invalid-resolved-at",
			freeze.Incident{
				Target:     "foo:bar",
				Status:     "FAILURE",
				Message:    "this is also failure",
				CausedAt:   "2001-02-03T00:01:00Z",
				ResolvedAt: "this is not a time",
			},
			Incident{},
			`resolved time is invalid: parsing time "this is not a time" as "2006-01-02T15:04:05Z07:00": cannot parse "this is not a time" as "2006"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, err := convertIncident(tt.Input)
			if err != nil {
				if err.Error() != tt.Error {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			} else if tt.Error != "" {
				t.Fatalf("expected error but got nil")
			}

			got := fmt.Sprint(output)
			want := fmt.Sprint(tt.Output)

			if got != want {
				t.Errorf("unexpected output:\nwant: %v\n got: %v", want, got)
			}
		})
	}
}

func TestConvertProbeHistory(t *testing.T) {
	tests := []struct {
		Name   string
		Input  freeze.ProbeHistory
		Output []Record
		Error  string
	}{
		{
			Name: "valid",
			Input: freeze.ProbeHistory{
				Target: "foo:bar",
				History: []freeze.Record{
					{
						CheckedAt: "2001-02-03T00:00:01Z",
						Status:    "HEALTHY",
						Message:   "foobar",
						Latency:   123.456,
					},
					{
						CheckedAt: "2001-02-03T00:00:02Z",
						Status:    "UNKNOWN",
						Message:   "foo bar",
						Latency:   234.567,
					},
				},
			},
			Output: []Record{
				{
					Target:    &url.URL{Scheme: "foo", Opaque: "bar"},
					CheckedAt: time.Date(2001, 2, 3, 0, 0, 1, 0, time.UTC),
					Status:    StatusHealthy,
					Message:   "foobar",
					Latency:   123456 * time.Microsecond,
				},
				{
					Target:    &url.URL{Scheme: "foo", Opaque: "bar"},
					CheckedAt: time.Date(2001, 2, 3, 0, 0, 2, 0, time.UTC),
					Status:    StatusUnknown,
					Message:   "foo bar",
					Latency:   234567 * time.Microsecond,
				},
			},
			Error: "",
		},
		{
			Name: "include-no-data",
			Input: freeze.ProbeHistory{
				Target: "foo:bar",
				History: []freeze.Record{
					{
						Status: "NO_DATA",
					},
					{
						CheckedAt: "2001-02-03T00:00:02Z",
						Status:    "UNKNOWN",
						Message:   "foo bar",
						Latency:   234.567,
					},
				},
			},
			Output: []Record{
				{
					Target:    &url.URL{Scheme: "foo", Opaque: "bar"},
					CheckedAt: time.Date(2001, 2, 3, 0, 0, 2, 0, time.UTC),
					Status:    StatusUnknown,
					Message:   "foo bar",
					Latency:   234567 * time.Microsecond,
				},
			},
			Error: "",
		},
		{
			Name: "invalid-target",
			Input: freeze.ProbeHistory{
				Target: "::invalid::",
				History: []freeze.Record{
					{
						CheckedAt: "2001-02-03T00:00:01Z",
						Status:    "HEALTHY",
						Message:   "foobar",
						Latency:   123.456,
					},
					{
						CheckedAt: "2001-02-03T00:00:02Z",
						Status:    "UNKNOWN",
						Message:   "foo bar",
						Latency:   234.567,
					},
				},
			},
			Output: []Record{},
			Error:  `invalid target URL: parse "::invalid::": missing protocol scheme`,
		},
		{
			Name: "invalid-checked-at",
			Input: freeze.ProbeHistory{
				Target: "foo:bar",
				History: []freeze.Record{
					{
						CheckedAt: "2001-02-03T00:00:01Z",
						Status:    "HEALTHY",
						Message:   "foobar",
						Latency:   123.456,
					},
					{
						CheckedAt: "this is not at timestamp",
						Status:    "UNKNOWN",
						Message:   "foo bar",
						Latency:   234.567,
					},
				},
			},
			Output: []Record{},
			Error:  `checked time is invalid: parsing time "this is not at timestamp" as "2006-01-02T15:04:05Z07:00": cannot parse "this is not at timestamp" as "2006"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, err := convertProbeHistory(tt.Input)
			if err != nil {
				if err.Error() != tt.Error {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			} else if tt.Error != "" {
				t.Fatalf("expected error but got nil")
			}

			got := fmt.Sprint(output)
			want := fmt.Sprint(tt.Output)

			if got != want {
				t.Errorf("unexpected output:\nwant: %v\n got: %v", want, got)
			}
		})
	}
}

var (
	DummyResponse = Response{freeze.Status{
		CurrentStatus: []freeze.ProbeHistory{
			{
				Target: "foo:bar",
				History: []freeze.Record{
					{
						CheckedAt: "2001-02-03T00:00:01Z",
						Status:    "HEALTHY",
						Message:   "foobar",
						Latency:   123.456,
					},
					{
						CheckedAt: "2001-02-03T00:00:02Z",
						Status:    "UNKNOWN",
						Message:   "foo bar",
						Latency:   234.567,
					},
				},
			},
			{
				Target: "hoge:fuga",
				History: []freeze.Record{
					{
						CheckedAt: "2001-02-03T00:00:01Z",
						Status:    "HEALTHY",
						Message:   "hello world",
						Latency:   123.456,
					},
				},
			},
		},
		CurrentIncidents: []freeze.Incident{
			{
				Target:   "foo:bar",
				Status:   "FAILURE",
				Message:  "this is failure",
				CausedAt: "2001-02-03T00:02:00Z",
			},
		},
		IncidentHistory: []freeze.Incident{
			{
				Target:     "foo:bar",
				Status:     "FAILURE",
				Message:    "this is also failure",
				CausedAt:   "2001-02-03T00:01:00Z",
				ResolvedAt: "2001-02-03T00:02:00Z",
			},
		},
	}}
)

func TestRecord_CurrentIncidents(t *testing.T) {
	incidents, err := DummyResponse.CurrentIncidents()
	if err != nil {
		t.Fatalf("failed to parse incident: %s", err)
	}

	if len(incidents) != 1 {
		t.Errorf("unexpected number of incidents: %#v", incidents)
	}

	if incidents[0].Message != "this is failure" {
		t.Errorf("unexpected message: %#v", incidents[0].Message)
	}
}

func TestRecord_IncidentHistories(t *testing.T) {
	incidents, err := DummyResponse.IncidentHistory()
	if err != nil {
		t.Fatalf("failed to parse incident: %s", err)
	}

	if len(incidents) != 1 {
		t.Errorf("unexpected number of incidents: %#v", incidents)
	}

	if incidents[0].Message != "this is also failure" {
		t.Errorf("unexpected message: %#v", incidents[0].Message)
	}
}

func TestRecord_Targets(t *testing.T) {
	targets, err := DummyResponse.Targets()
	if err != nil {
		t.Fatalf("failed to parse targets: %s", err)
	}

	if len(targets) != 2 {
		t.Errorf("unexpected number of targets: %#v", targets)
	}

	x := targets[0].String() + ", " + targets[1].String()

	if x != "foo:bar, hoge:fuga" && x != "hoge:fuga, foo:bar" {
		t.Errorf("unexpected targets: %s", x)
	}
}

func TestRecord_RecordsOf(t *testing.T) {
	records, err := DummyResponse.RecordsOf(&url.URL{Scheme: "foo", Opaque: "bar"})
	if err != nil {
		t.Fatalf("failed to parse records: %s", err)
	}

	if len(records) != 2 {
		t.Errorf("unexpected number of records: %#v", records)
	}

	if records[0].Message != "foobar" {
		t.Errorf("unexpected message of records[0]: %#v", records[0])
	}

	if records[1].Message != "foo bar" {
		t.Errorf("unexpected message of records[1]: %#v", records[1])
	}

	records, err = DummyResponse.RecordsOf(&url.URL{Scheme: "hoge", Opaque: "fuga"})
	if err != nil {
		t.Fatalf("failed to parse records: %s", err)
	}

	if len(records) != 1 {
		t.Errorf("unexpected number of records: %#v", records)
	}
}

func TestRecord_AllRecords(t *testing.T) {
	records, err := DummyResponse.AllRecords()
	if err != nil {
		t.Fatalf("failed to parse records: %s", err)
	}

	if len(records) != 2 {
		t.Fatalf("unexpected number of record lists: %#v", records)
	}
}
