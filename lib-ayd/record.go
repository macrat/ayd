package ayd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/macrat/ayd/internal/ayderr"
)

// Record is a record in Ayd log
type Record struct {
	CheckedAt time.Time
	Status    Status
	Latency   time.Duration
	Target    *URL
	Message   string
	Extra     map[string]interface{}
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

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Record) UnmarshalJSON(data []byte) error {
	*r = Record{}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ayderr.New(ErrInvalidRecord, err, "invalid record")
	}

	var err error

	if value, ok := raw["time"]; !ok {
		return ayderr.New(ErrInvalidRecord, nil, "invalid record: time: missing required field")
	} else {
		if s, ok := value.(string); !ok {
			return ayderr.New(ErrInvalidRecord, nil, "invalid record: time: should be a string")
		} else if r.CheckedAt, err = time.Parse(time.RFC3339, s); err != nil {
			return ayderr.New(ErrInvalidRecord, err, "invalid record: time")
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

	_, err := fmt.Fprintf(
		head,
		`{"time":"%s","status":"%s","latency":%v,"target":%q`,
		r.CheckedAt.Format(time.RFC3339),
		r.Status,
		float64(r.Latency.Microseconds())/1000,
		r.Target,
	)
	if err != nil {
		return nil, err
	}

	enc := json.NewEncoder(head)

	if r.Message != "" {
		if _, err = head.Write([]byte(`,"message":`)); err != nil {
			return nil, err
		}
		if err = enc.Encode(r.Message); err != nil {
			return nil, err
		}
		head.Truncate(head.Len() - 1) // drop newline
	}

	for k, v := range r.Extra {
		if _, err = head.Write([]byte(",")); err != nil {
			return nil, err
		}
		if err = enc.Encode(k); err != nil {
			return nil, err
		}
		head.Truncate(head.Len() - 1) // drop newline
		if _, err = head.Write([]byte(":")); err != nil {
			return nil, err
		}
		if err = enc.Encode(v); err != nil {
			return nil, err
		}
		head.Truncate(head.Len() - 1) // drop newline
	}

	_, err = head.Write([]byte("}"))

	return head.Bytes(), err
}
