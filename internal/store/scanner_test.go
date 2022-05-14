package store_test

import (
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/store"
	"github.com/macrat/ayd/internal/testutil"
)

func TestStore_OpenLog(t *testing.T) {
	inMemory, err := store.New("", io.Discard)
	if err != nil {
		t.Fatalf("failed to create in-memory store: %s", err)
	}
	defer inMemory.Close()
	inStorage := testutil.NewStoreWithLog(t)

	stores := []struct {
		Name  string
		Store *store.Store
	}{
		{"in-memory", inMemory},
		{"storage", inStorage},
	}

	if scanner, err := inStorage.OpenLog(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Now()); err != nil {
		t.Fatalf("failed to prepare in-memory store: %s", err)
	} else {
		for scanner.Scan() {
			inMemory.Report(scanner.Record().Target, scanner.Record())
		}
	}

	for _, s := range stores {
		t.Run(s.Name, func(t *testing.T) {
			s := s.Store

			tests := []struct {
				Since, Until time.Time
				Messages     []string
			}{
				{
					time.Date(2021, 1, 2, 15, 4, 0, 0, time.UTC),
					time.Date(2021, 1, 2, 15, 4, 10, 0, time.UTC),
					[]string{
						"hello world",
						"this is failure",
						"hello world!",
						"this is healthy",
						"hello world!!",
						"this is aborted",
						"this is unknown",
					},
				},
				{
					time.Date(2021, 1, 2, 15, 4, 6, 0, time.UTC),
					time.Date(2021, 1, 2, 15, 4, 8, 0, time.UTC),
					[]string{
						"hello world!",
						"this is healthy",
						"hello world!!",
					},
				},
				{
					time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
					nil,
				},
			}

			for _, tt := range tests {
				t.Run(tt.Since.Format(time.RFC3339), func(t *testing.T) {
					scanner, err := s.OpenLog(tt.Since, tt.Until)
					if err != nil {
						t.Fatalf("failed to open scanner: %s", err)
					}

					var actual []string
					for scanner.Scan() {
						actual = append(actual, scanner.Record().Message)
					}

					if diff := cmp.Diff(tt.Messages, actual); diff != "" {
						t.Error(diff)
					}
				})
			}
		})
	}
}
