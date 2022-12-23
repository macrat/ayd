package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

// searchLog seeks and searches a log file roughly.
// You should search the time at least 1 hour before from your actual target time, because the record ordering in log files might swap maximum 1 hour.
//
// The result can contains error up to `accuracy` bytes.
// If the `accuracy` is 0, the result is perfectly aligned to the begin of the found record.
func searchLog(f io.ReadSeeker, target time.Time, accuracy int64) error {
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if size <= accuracy {
		_, err := f.Seek(0, io.SeekStart)
		return err
	}

	right := size
	var left int64
	mid := right / 2

	buf := make([]byte, 256)

	var rec struct {
		Time string `json:"time"`
	}

	for left+accuracy < right {
		if _, err := f.Seek(mid, io.SeekStart); err != nil {
			return err
		}

		n, err := f.Read(buf[:])
		if errors.Is(err, io.EOF) {
			right = left + (right-left)/2
			continue
		} else if err != nil {
			return err
		}
		line := buf[:n]
		if idx := bytes.IndexRune(line, '\n'); idx < 0 {
			buf = make([]byte, len(buf)*2)
			continue
		} else if idx == 0 {
			mid += 1
			continue
		} else {
			line = line[:idx]
		}

		if line[0] == '{' {
			var t time.Time
			err = json.Unmarshal(line, &rec)
			if err == nil {
				t, err = api.ParseTime(rec.Time)
			}

			if err == nil {
				if target.After(t) {
					left = mid + int64(len(line)) + 1
				} else {
					right = left + (right-left)/2
				}

				mid = left + (right-left)/2
				continue
			}
		}

		// The current position is not valid JSON.
		mid += int64(len(line)) + 1
		if mid >= right {
			right = left + (right-left)/2
			mid = left + (right-left)/2
		}
	}

	if _, err := f.Seek(left, io.SeekStart); err != nil {
		return err
	}

	return nil
}
