package ayd

import (
	"bytes"
	"encoding/json"
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

type jsonRecord struct {
	CheckedAt string  `json:"time"`
	Status    Status  `json:"status"`
	Latency   float64 `json:"latency"`
	Target    *URL    `json:"target"`
	Message   string  `json:"message,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Record) UnmarshalJSON(data []byte) error {
	var jr jsonRecord

	if err := json.Unmarshal(data, &jr); err != nil {
		return ayderr.New(ErrInvalidRecord, err, "invalid record")
	}

	checkedAt, err := time.Parse(time.RFC3339, jr.CheckedAt)
	if err != nil {
		return ayderr.New(ErrInvalidRecord, err, "invalid record")
	}

	var extra map[string]interface{}
	if err := json.Unmarshal(data, &extra); err != nil {
		return ayderr.New(ErrInvalidRecord, err, "invalid record")
	}
	for key := range extra {
		switch key {
		case "time", "status", "latency", "target", "message":
			delete(extra, key)
		}
	}

	*r = Record{
		CheckedAt: checkedAt,
		Status:    jr.Status,
		Target:    jr.Target,
		Latency:   time.Duration(jr.Latency * float64(time.Millisecond)),
		Message:   jr.Message,
		Extra:     extra,
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (r Record) MarshalJSON() ([]byte, error) {
	var head bytes.Buffer

	err := json.NewEncoder(&head).Encode(jsonRecord{
		CheckedAt: r.CheckedAt.Format(time.RFC3339),
		Status:    r.Status,
		Latency:   float64(r.Latency.Microseconds()) / 1000,
		Target:    r.Target,
		Message:   r.Message,
	})
	if err != nil {
		return nil, err
	}

	if len(r.Extra) > 0 {
		head.Truncate(len(head.Bytes()) - 2) // drop newline and last "}".
		head.Write([]byte(","))

		var tail bytes.Buffer
		err = json.NewEncoder(&tail).Encode(r.Extra)
		if err != nil {
			return nil, err
		}
		tail.ReadByte()
		_, err = tail.WriteTo(&head)
		if err != nil {
			return nil, err
		}
	}

	head.Truncate(len(head.Bytes()) - 1) // drop newline

	return head.Bytes(), nil
}
