package ayd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/macrat/ayd/store/freeze"
)

var (
	ErrNoSuchTarget = errors.New("no such target")
)

func convertIncident(i freeze.Incident) (Incident, error) {
	target, err := url.Parse(i.Target)
	if err != nil {
		return Incident{}, fmt.Errorf("invalid target URL: %w", err)
	}

	causedAt, err := time.Parse(time.RFC3339, i.CausedAt)
	if err != nil {
		return Incident{}, fmt.Errorf("caused time is invalid: %w", err)
	}

	var resolvedAt time.Time
	if i.ResolvedAt != "" {
		resolvedAt, err = time.Parse(time.RFC3339, i.ResolvedAt)
		if err != nil {
			return Incident{}, fmt.Errorf("resolved time is invalid: %w", err)
		}
	}

	return Incident{
		Target:     target,
		Status:     ParseStatus(i.Status),
		Message:    i.Message,
		CausedAt:   causedAt,
		ResolvedAt: resolvedAt,
	}, nil
}

func convertRecord(target *url.URL, r freeze.Record) (Record, error) {
	checkedAt, err := time.Parse(time.RFC3339, r.CheckedAt)
	if err != nil {
		return Record{}, fmt.Errorf("checked time is invalid: %w", err)
	}

	return Record{
		CheckedAt: checkedAt,
		Status:    ParseStatus(r.Status),
		Latency:   time.Duration(r.Latency * float64(time.Millisecond)),
		Target:    target,
		Message:   r.Message,
	}, nil
}

func convertProbeHistory(h freeze.ProbeHistory) ([]Record, error) {
	target, err := url.Parse(h.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	rs := make([]Record, 0, len(h.History))

	for _, x := range h.History {
		if x.Status == "NO_DATA" {
			continue
		}
		r, err := convertRecord(target, x)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}

	return rs, nil
}

// Response is a response from Ayd server
type Response struct {
	status freeze.Status
}

// CurrentIncidents returns list of Incident that current causing
func (r Response) CurrentIncidents() ([]Incident, error) {
	rs := make([]Incident, len(r.status.CurrentIncidents))

	var err error
	for i, x := range r.status.CurrentIncidents {
		rs[i], err = convertIncident(x)
		if err != nil {
			return nil, err
		}
	}

	return rs, nil
}

// IncidentHistory returns list of Incident that already resolved
//
// If you want get current causing incidents, please use CurrentIncidents method.
func (r Response) IncidentHistory() ([]Incident, error) {
	rs := make([]Incident, len(r.status.IncidentHistory))

	var err error
	for i, x := range r.status.IncidentHistory {
		rs[i], err = convertIncident(x)
		if err != nil {
			return nil, err
		}
	}

	return rs, nil
}

// Targets returns target URLs that to status checking
func (r Response) Targets() ([]*url.URL, error) {
	rs := make([]*url.URL, len(r.status.CurrentStatus))

	var err error
	for i, x := range r.status.CurrentStatus {
		rs[i], err = url.Parse(x.Target)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL: %w", err)
		}
	}

	return rs, nil
}

// RecordsOf returns Record history of specified target by argument
func (r Response) RecordsOf(target *url.URL) ([]Record, error) {
	targetStr := target.String()
	for _, x := range r.status.CurrentStatus {
		if x.Target == targetStr {
			return convertProbeHistory(x)
		}
	}

	return nil, fmt.Errorf("%s: %w", targetStr, ErrNoSuchTarget)
}

// AllRecords returns Record history of all targets
func (r Response) AllRecords() ([][]Record, error) {
	rs := make([][]Record, len(r.status.CurrentStatus))

	var err error
	for i, x := range r.status.CurrentStatus {
		rs[i], err = convertProbeHistory(x)
		if err != nil {
			return nil, err
		}
	}

	return rs, nil
}

// Fetch is fetch Ayd json API and returns Response
func Fetch(u *url.URL) (Response, error) {
	var err error
	u, err = u.Parse("status.json")
	if err != nil {
		return Response{}, fmt.Errorf("failed to parse URL: %w", err)
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return Response{}, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("failed to read response: %w", err)
	}

	var r Response
	err = json.Unmarshal(raw, &r.status)
	if err != nil {
		return Response{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return r, nil
}
