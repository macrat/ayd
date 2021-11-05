package scheme_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestAlertSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name     string
		URLs     []string
		Messages []string
		Error    string
	}{
		{"empty", []string{}, []string{}, ""},
		{"single", []string{"dummy:?message=abc"}, []string{"abc"}, ""},
		{"multiple", []string{"dummy:?message=abc", "dummy:?message=def"}, []string{"abc", "def"}, ""},
		{"invalid", []string{"dummy:#its_okay", "::invalid::", "no.such:abc", "dummy:#its_also_okay"}, nil, "invalid alert URL:\n  ::invalid::: invalid URL\n  no.such:abc: unsupported scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			as, err := scheme.NewAlertSet(tt.URLs)
			if tt.Error != "" {
				if err == nil {
					t.Fatalf("expected error but returns nil")
				}
				if err.Error() != tt.Error {
					t.Errorf("unexpected error\n--- expected ---\n%s\n--- but got ---\n%s", tt.Error, err)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error\n%s", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			rec := api.Record{
				Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
				Status:    api.StatusFailure,
				CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
				Message:   "foobar",
			}

			r := &testutil.DummyReporter{}

			as.Alert(ctx, r, rec)

			if len(r.Records) != len(tt.Messages) {
				t.Fatalf("expected %d records but got %d records", len(tt.Messages), len(r.Records))
			}

			for _, expect := range tt.Messages {
				ok := false
				for _, found := range r.Records {
					if found.Message == expect {
						ok = true
						break
					}
				}

				if !ok {
					t.Errorf("expected message %#v was not found", expect)
				}
			}
		})
	}
}

func TestAlertSet_blocking(t *testing.T) {
	t.Parallel()

	as, err := scheme.NewAlertSet([]string{"dummy:?latency=500ms", "dummy:?latency=1000ms"})
	if err != nil {
		t.Fatalf("failed to create a new set: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rec := api.Record{
		Target:    &url.URL{Scheme: "dummy", Opaque: "failure"},
		Status:    api.StatusFailure,
		CheckedAt: time.Date(2001, 2, 3, 16, 5, 6, 0, time.UTC),
		Message:   "foobar",
	}

	r := &testutil.DummyReporter{}

	stime := time.Now()
	as.Alert(ctx, r, rec)
	delay := time.Now().Sub(stime)

	if len(r.Records) != 2 {
		t.Errorf("unexpected number of records\n%v", r.Records)
	}

	if delay < 1*time.Second {
		t.Errorf("expected to blocking during alert function running but returns too fast: %s", delay)
	}
}
