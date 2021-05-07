package ayd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Record is a record in Ayd log
type Record struct {
	// CheckedAt is the time the check started
	CheckedAt time.Time

	Status Status

	Latency time.Duration

	Target *url.URL

	// Message is the reason of the status, or extra informations of the check
	Message string
}

func unescapeMessage(s string) string {
	for _, x := range []struct {
		From string
		To   string
	}{
		{`\t`, "\t"},
		{`\n`, "\n"},
		{`\\`, `\`},
	} {
		s = strings.ReplaceAll(s, x.From, x.To)
	}
	return s
}

// ParseRecord is parse string as a Record row in the log
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
	r.Status.UnmarshalText([]byte(ss[1]))
	latency, err := strconv.ParseFloat(ss[2], 64)
	if err != nil {
		return Record{}, err
	}
	target = ss[3]
	r.Message = unescapeMessage(ss[4])

	r.CheckedAt, err = time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return Record{}, err
	}

	r.Latency = time.Duration(latency * float64(time.Millisecond))

	r.Target, err = url.Parse(target)
	if err != nil {
		return Record{}, err
	}

	if (r.Target.Scheme == "exec" || r.Target.Scheme == "source") && r.Target.Opaque == "" {
		r.Target.Opaque = r.Target.Path
		r.Target.Path = ""
	}

	return r, nil
}

func escapeMessage(s string) string {
	for _, x := range []struct {
		From string
		To   string
	}{
		{`\`, `\\`},
		{"\t", `\t`},
		{"\n", `\n`},
	} {
		s = strings.ReplaceAll(s, x.From, x.To)
	}
	return s
}

// String is make Result a string for row in the log
func (r Record) String() string {
	return strings.Join([]string{
		r.CheckedAt.Format(time.RFC3339),
		r.Status.String(),
		strconv.FormatFloat(float64(r.Latency.Microseconds())/1000, 'f', 3, 64),
		r.Target.Redacted(),
		escapeMessage(r.Message),
	}, "\t")
}
