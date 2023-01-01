package main

import (
	"time"
)

func init() {
	CurrentTime = func() time.Time {
		return time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC)
	}
}
