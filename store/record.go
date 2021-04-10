package store

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	CheckedAt time.Time
	Target    *url.URL
	Status    Status
	Message   string
	Latency   time.Duration
}

func ParseRecord(s string) (Record, error) {
	var r Record
	var timestamp string
	var latency float64
	var target string

	ss := strings.SplitN(s, "\t", 5)
	if len(ss) != 5 {
		return Record{}, fmt.Errorf("unexpected column count")
	}

	timestamp = ss[0]
	r.Status = ParseStatus(ss[1])
	latency, err := strconv.ParseFloat(ss[2], 64)
	if err != nil {
		return Record{}, err
	}
	target = ss[3]
	r.Message = ss[4]

	r.CheckedAt, err = time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return Record{}, err
	}

	r.Latency = time.Duration(latency * float64(time.Millisecond))

	r.Target, err = url.Parse(target)
	if err != nil {
		return Record{}, err
	}

	return r, nil
}

func (r Record) Sanitize() Record {
	return Record{
		CheckedAt: r.CheckedAt,
		Target:    r.Target,
		Status:    r.Status,
		Message:   strings.ReplaceAll(strings.ReplaceAll(r.Message, "\t", "    "), "\n", " "),
		Latency:   r.Latency,
	}
}

func (r Record) String() string {
	return strings.Join([]string{
		r.CheckedAt.Format(time.RFC3339),
		r.Status.String(),
		fmt.Sprintf("%.3f", float64(r.Latency.Microseconds())/1000),
		r.Target.String(),
		r.Message,
	}, "\t")
}
