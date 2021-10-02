package ayd

import (
	"encoding/json"
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

// UnmarshalText is unmarshal from text
func (r *Record) UnmarshalText(text []byte) (err error) {
	*r, err = ParseRecord(string(text))
	return
}

// MarshalText is marshal to text
func (r Record) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

type jsonRecord struct {
	CheckedAt time.Time `json:"checked_at"`
	Status    Status    `json:"status"`
	Latency   float64   `json:"latency"`
	Target    string    `json:"target"`
	Message   string    `json:"message"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Record) UnmarshalJSON(data []byte) error {
	var jr jsonRecord

	if err := json.Unmarshal(data, &jr); err != nil {
		return err
	}

	target, err := url.Parse(jr.Target)
	if err != nil {
		return err
	}

	*r = Record{
		CheckedAt: jr.CheckedAt,
		Status:    jr.Status,
		Target:    target,
		Latency:   time.Duration(jr.Latency * float64(time.Millisecond)),
		Message:   jr.Message,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (r Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonRecord{
		CheckedAt: r.CheckedAt,
		Status:    r.Status,
		Latency:   float64(r.Latency.Microseconds()) / 1000,
		Target:    r.Target.String(),
		Message:   r.Message,
	})
}
