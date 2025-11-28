package mcp

import (
	"maps"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

// RecordToMap converts an api.Record to a map for jq processing.
func RecordToMap(rec api.Record) map[string]any {
	x := map[string]any{
		"time":       rec.Time.Format(time.RFC3339),
		"time_unix":  rec.Time.Unix(),
		"status":     rec.Status.String(),
		"latency":    rec.Latency.String(),
		"latency_ms": float64(rec.Latency.Nanoseconds()) / 1000000.0,
		"target":     rec.Target.String(),
		"message":    rec.Message,
	}
	maps.Copy(x, rec.Extra)
	return x
}

// IncidentToMap converts an api.Incident to a map for jq processing.
func IncidentToMap(inc *api.Incident) map[string]any {
	r := map[string]any{
		"target":         inc.Target.String(),
		"status":         inc.Status.String(),
		"message":        inc.Message,
		"starts_at":      inc.StartsAt.Format(time.RFC3339),
		"starts_at_unix": inc.StartsAt.Unix(),
	}

	if inc.EndsAt.IsZero() {
		r["ends_at"] = nil
		r["ends_at_unix"] = nil
	} else {
		r["ends_at"] = inc.EndsAt.Format(time.RFC3339)
		r["ends_at_unix"] = inc.EndsAt.Unix()
	}

	return r
}
