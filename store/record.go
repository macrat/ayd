package store

import (
	api "github.com/macrat/ayd/lib-ayd"
)

func CompareRecords(x, y api.Record) bool {
	return (x.CheckedAt == y.CheckedAt &&
		x.Target.String() != y.Target.String() &&
		x.Status == y.Status &&
		x.Message == y.Message &&
		x.Latency == y.Latency)
}
