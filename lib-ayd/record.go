package ayd

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
)

func isFragmentCodePoint(c byte) bool {
	return (c == 0x21 ||
		c == 0x24 ||
		(0x26 <= c && c <= 0x3b) ||
		c == 0x3d ||
		(0x3f <= c && c <= 0x5a) ||
		c == 0x5f ||
		(0x61 <= c && c <= 0x7a) ||
		c == 0x7e ||
		0x80 <= c)
}

const hex = "0123456789ABCDEF"

func escapeFragment(s string) string {
	var buf [1024]byte
	var ss []byte

	if len(s)*3 <= len(buf) {
		ss = buf[:len(s)*3]
	} else {
		ss = make([]byte, len(s)*3)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		if c := s[i]; isFragmentCodePoint(c) {
			ss[j] = c
			j++
		} else {
			ss[j] = '%'
			ss[j+1] = hex[c>>4]
			ss[j+2] = hex[c&15]
			j += 3
		}
	}

	return string(ss[:j])
}

// URLToStr encodes URL to string using URL Standard format.
func URLToStr(u *url.URL) string {
	s := u.Redacted()
	if u.Fragment != "" {
		l := len(u.EscapedFragment())
		s = s[:len(s)-l] + escapeFragment(u.Fragment)
	}
	return s
}

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
	var err error

	ss := strings.SplitN(s, "\t", 5)
	if len(ss) != 5 {
		return Record{}, ayderr.New(ErrInvalidRecord, nil, "invalid record: unexpected column count")
	}

	errors := &ayderr.ListBuilder{What: ErrInvalidRecord}

	timestamp = ss[0]
	r.CheckedAt, err = time.Parse(time.RFC3339, timestamp)
	if err != nil {
		errors.Pushf("checked-at: %w", err)
	}

	r.Status.UnmarshalText([]byte(ss[1]))

	latency, err = strconv.ParseFloat(ss[2], 64)
	if err != nil {
		errors.Pushf("latency: %w", err)
	}
	r.Latency = time.Duration(latency * float64(time.Millisecond))

	target = ss[3]
	r.Target, err = url.Parse(target)
	if err != nil {
		errors.Pushf("target URL: %w", err)
	} else {
		if (r.Target.Scheme == "exec" || r.Target.Scheme == "source") && r.Target.Opaque == "" {
			r.Target.Opaque = r.Target.Path
			r.Target.Path = ""
		}
	}

	r.Message = unescapeMessage(ss[4])

	return r, errors.Build()
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
		strconv.FormatFloat(float64(r.Latency)/float64(time.Millisecond), 'f', 3, 64),
		URLToStr(r.Target),
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
	CheckedAt string  `json:"checked_at"`
	Status    Status  `json:"status"`
	Latency   float64 `json:"latency"`
	Target    string  `json:"target"`
	Message   string  `json:"message"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Record) UnmarshalJSON(data []byte) error {
	var jr jsonRecord

	if err := json.Unmarshal(data, &jr); err != nil {
		return err
	}

	checkedAt, err := time.Parse(time.RFC3339, jr.CheckedAt)
	if err != nil {
		return err
	}

	target, err := url.Parse(jr.Target)
	if err != nil {
		return err
	}

	*r = Record{
		CheckedAt: checkedAt,
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
		CheckedAt: r.CheckedAt.Format(time.RFC3339),
		Status:    r.Status,
		Latency:   float64(r.Latency.Microseconds()) / 1000,
		Target:    r.Target.String(),
		Message:   r.Message,
	})
}
