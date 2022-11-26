package store

import (
	"errors"
	"sync"
)

var (
	// ErrLogUnmatch is an error causes when the log index can't be used because log file has updated.
	ErrLogUnmatch = errors.New("error: log is unmatch to the index")
)

// indexPeriod is a entry of [indexer].
type indexPeriod struct {
	Start int64 // Start position in the log file in bytes.
	End   int64 // End position in the log file in bytes.
	Since int64 // Minimal timestamp in UNIX time that included in this period.
	Until int64 // Maximum timestamp in UNIX time that included in this period.
	Size  int64 // Number of log records in this period.
}

// indexer is the index for make faster to read log file.
type indexer struct {
	sync.Mutex

	periods  []indexPeriod
	interval int64
}

// newIndexer creates a new [indexer].
func newIndexer() *indexer {
	return &indexer{
		periods:  make([]indexPeriod, 1),
		interval: 10000,
	}
}

// AppendEntry stores new entry to the [indexer].
//
// `start` is the head position of record in log file.
// `end` is the tail position of record in log file.
// `time` is the UNIX time of the log entry.
//
// The `start` should equal to the previous entry's `end`. Otherwise, this method returns error because log file could has been updated or rotated.
func (idx *indexer) AppendEntry(start, end, time int64) error {
	idx.Lock()
	defer idx.Unlock()

	return idx.AppendEntryWithoutLock(start, end, time)
}

// AppendEntryWithoutLock stores new entry to the [indexer] without lock mutex.
//
// The arguments are the same as AppendEntryWithoutLock.
func (idx *indexer) AppendEntryWithoutLock(start, end, time int64) error {
	i := len(idx.periods) - 1

	if idx.periods[i].End != start {
		return ErrLogUnmatch
	}

	if idx.periods[i].Size == 0 {
		idx.periods[i].Start = start
		idx.periods[i].End = end
		idx.periods[i].Since = time
		idx.periods[i].Until = time
		idx.periods[i].Size = 1
	} else if idx.periods[i].Size < idx.interval {
		if idx.periods[i].Since > time {
			idx.periods[i].Since = time
		}
		if idx.periods[i].Until < time {
			idx.periods[i].Until = time
		}
		idx.periods[i].Size++
		idx.periods[i].End = end
	} else {
		idx.periods = append(idx.periods, indexPeriod{
			Start: start,
			End:   end,
			Since: time,
			Until: time,
			Size:  1,
		})
	}

	return nil
}

// AppendInvalidRangeWithoutLock records a range of log file that doesn't contain valid log entry, without lock mutex.
//
// `start` and `end` of arguments are the same meaning to [indexer.AppendEntry].
func (idx *indexer) AppendInvalidRangeWithoutLock(start, end int64) error {
	i := len(idx.periods) - 1

	if idx.periods[i].End != start {
		return ErrLogUnmatch
	}

	if idx.periods[i].Size == 0 {
		idx.periods[i].End = end
	} else {
		idx.periods = append(idx.periods, indexPeriod{
			Start: start,
			End:   end,
		})
	}

	return nil
}

// Search picks up the ranges in log file that includes specified period by arguments.
func (idx *indexer) Search(since, until int64) []logRange {
	idx.Lock()
	defer idx.Unlock()

	var ranges []logRange

	for _, x := range idx.periods {
		if x.Since <= until && since <= x.Until && x.Size != 0 {
			if len(ranges) == 0 || ranges[len(ranges)-1].End != x.Start {
				ranges = append(ranges, logRange{
					Start: x.Start,
					End:   x.End,
					Size:  x.Size,
				})
			} else {
				ranges[len(ranges)-1].End = x.End
				ranges[len(ranges)-1].Size += x.Size
			}
		}
	}

	return ranges
}

// logRange is a range in log file.
type logRange struct {
	Start int64 // Start position in bytes.
	End   int64 // End position in bytes.
	Size  int64 // Number of included log entries.
}

// Reset resets indexer.
func (idx *indexer) Reset() {
	idx.Lock()
	defer idx.Unlock()

	idx.ResetWithoutLock()
}

// ResetWithoutLock resets indexer without lock mutex.
func (idx *indexer) ResetWithoutLock() {
	idx.periods = make([]indexPeriod, 1)
}
