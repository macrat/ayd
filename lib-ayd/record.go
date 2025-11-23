package ayd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/macrat/ayd/internal/ayderr"
)

func isReservedKey(key string) bool {
	return key == "time" || key == "status" || key == "latency" || key == "target"
}

// Record is a record in Ayd log
type Record struct {
	Time    time.Time
	Status  Status
	Latency time.Duration
	Target  *URL
	Message string
	Extra   map[string]interface{}
}

// ParseRecord is parse string as a Record row in the log
func ParseRecord(s string) (Record, error) {
	var r Record
	return r, r.UnmarshalJSON([]byte(s))
}

// String is make Result a string for row in the log
func (r Record) String() string {
	b, err := r.MarshalJSON()
	if err != nil {
		return `{"error":"invalid record"}`
	}
	return string(b)
}

// ReadableExtra returns Extra map but encode each values as string.
func (r Record) ReadableExtra() []ExtraPair {
	if len(r.Extra) == 0 {
		return nil
	}

	xs := make([]ExtraPair, 0, len(r.Extra))
	for k, v := range r.Extra {
		if isReservedKey(k) {
			continue
		}

		s := ""
		switch x := v.(type) {
		case string:
			s = x
		case int, int32, int64, float32, float64:
			s = fmt.Sprint(x)
		default:
			b, err := json.Marshal(x)
			if err != nil {
				s = fmt.Sprint(x)
			} else {
				s = string(b)
			}
		}
		xs = append(xs, ExtraPair{k, s})
	}

	sort.Slice(xs, func(i, j int) bool {
		return xs[i].Key < xs[j].Key
	})

	return xs
}

// ExtraPair is a pair of string, for Record.StringExtra..
type ExtraPair struct {
	Key   string
	Value string
}

// ReadableMessage returns human readable message of the message field and extra fields.
func (r Record) ReadableMessage() string {
	if len(r.Extra) == 0 {
		return r.Message
	}
	var buf strings.Builder
	buf.WriteString(r.Message)
	if len(r.Message) > 0 && r.Message[len(r.Message)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString("---")
	for _, e := range r.ReadableExtra() {
		if !strings.Contains(e.Value, "\n") {
			fmt.Fprintf(&buf, "\n%s: %s", e.Key, e.Value)
		} else {
			fmt.Fprintf(&buf, "\n%s: |", e.Key)
			for _, line := range strings.Split(e.Value, "\n") {
				fmt.Fprintf(&buf, "\n  %s", line)
			}
		}
	}
	return buf.String()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Record) UnmarshalJSON(data []byte) (err error) {
	*r = Record{}

	var raw map[string]interface{}
	if err = json.Unmarshal(data, &raw); err != nil {
		return ayderr.New(ErrInvalidRecord, err, "invalid record")
	}

	if value, ok := raw["time"]; !ok {
		return ayderr.New(ErrInvalidRecord, nil, "invalid record: time: missing required field")
	} else {
		switch v := value.(type) {
		case float64:
			if v < 0 {
				r.Time = time.UnixMilli(0)
			} else {
				maxTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.Local)
				if v > float64(maxTime.Unix()) {
					r.Time = maxTime
				} else {
					r.Time = time.UnixMilli(int64(v * 1000))
				}
			}
		case string:
			if r.Time, err = ParseTime(v); err != nil {
				return ayderr.New(ErrInvalidRecord, err, "invalid record: time")
			}
		default:
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: time: should be a string or a number")
		}
		delete(raw, "time")
	}

	if value, ok := raw["status"]; ok {
		if s, ok := value.(string); !ok {
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: status: should be a string")
		} else {
			r.Status = ParseStatus(s)
		}
		delete(raw, "status")
	}

	if value, ok := raw["latency"]; ok {
		if f, ok := value.(float64); !ok {
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: latency: should be a number")
		} else {
			r.Latency = time.Duration(f * float64(time.Millisecond))
		}
		delete(raw, "latency")
	}

	if value, ok := raw["target"]; !ok {
		return ayderr.New(ErrInvalidRecord, nil, "invalid record: target: missing required field")
	} else {
		if s, ok := value.(string); !ok {
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: target: should be a string")
		} else if r.Target, err = ParseURL(s); err != nil {
			return ayderr.New(ErrInvalidRecord, err, "invalid record: target")
		}
		if r.Target.Scheme == "" {
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: target: parse %q: missing protocol scheme", value)
		}
		delete(raw, "target")
	}

	if value, ok := raw["message"]; ok {
		if s, ok := value.(string); !ok {
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: message: should be a string")
		} else {
			r.Message = s
		}
		delete(raw, "message")
	}

	if len(raw) > 0 {
		r.Extra = raw
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (r Record) MarshalJSON() ([]byte, error) {
	head := bytes.NewBuffer(make([]byte, 0, 256))

	target := ""
	if r.Target != nil {
		target = r.Target.String()
	}

	_, err := fmt.Fprintf(
		head,
		`{"time":"%s", "status":"%s", "latency":%.3f, "target":%q`,
		r.Time.Format(time.RFC3339),
		r.Status,
		float64(r.Latency.Microseconds())/1000,
		target,
	)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(head)

	if r.Message != "" {
		if _, err = head.Write([]byte(`, "message":`)); err != nil {
			return nil, err
		}
		if err = enc.Encode(r.Message); err != nil {
			return nil, err
		}
		head.Truncate(head.Len() - 1) // drop newline
	}

	extras := make([]extraPair, 0, len(r.Extra))
	for k, v := range r.Extra {
		if !isReservedKey(k) {
			extras = append(extras, extraPair{k, v})
		}
	}

	sort.Slice(extras, func(i, j int) bool {
		return extras[i].Key < extras[j].Key
	})

	for _, e := range extras {
		if _, err = head.Write([]byte(", ")); err != nil {
			return nil, err
		}
		if err = enc.Encode(e.Key); err != nil {
			return nil, err
		}
		head.Truncate(head.Len() - 1) // drop newline
		if _, err = head.Write([]byte(":")); err != nil {
			return nil, err
		}
		if err = enc.Encode(e.Value); err != nil {
			return nil, err
		}
		head.Truncate(head.Len() - 1) // drop newline
	}

	_, err = head.Write([]byte("}"))

	return head.Bytes(), err
}

type extraPair struct {
	Key   string
	Value interface{}
}
