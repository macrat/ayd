package ayd_test

import (
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/macrat/ayd/lib-ayd"
)

func TestIncident(t *testing.T) {
	assert := func(t *testing.T, i1, i2 ayd.Incident) {
		if i1.Target.String() != i2.Target.String() {
			t.Errorf("the target is different: %s != %s", i1.Target, i2.Target)
		}

		if i1.Status != i2.Status {
			t.Errorf("the status is different: %s != %s", i1.Status, i2.Status)
		}

		if i1.Message != i2.Message {
			t.Errorf("the message is different: %s != %s", i1.Message, i2.Message)
		}

		if i1.StartsAt.String() != i2.StartsAt.String() {
			t.Errorf("the starts_at is different: %s != %s", i1.StartsAt, i2.StartsAt)
		}

		if i1.EndsAt.String() != i2.EndsAt.String() {
			t.Errorf("the ends_at is different: %s != %s", i1.EndsAt, i2.EndsAt)
		}
	}

	t.Run("marshal-and-unmarshal", func(t *testing.T) {
		i1 := ayd.Incident{
			Target:   &ayd.URL{Scheme: "dummy", Opaque: "failure", Fragment: "hello-world"},
			Status:   ayd.StatusFailure,
			Message:  "it's incident",
			StartsAt: time.Date(2001, 1, 2, 15, 4, 5, 0, time.UTC),
			EndsAt:   time.Date(2021, 6, 5, 16, 3, 2, 0, time.UTC),
		}

		j, err := json.Marshal(i1)
		if err != nil {
			t.Fatalf("failed to marshal: %s", err)
		}

		var i2 ayd.Incident
		err = json.Unmarshal(j, &i2)
		if err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		assert(t, i1, i2)
	})

	t.Run("unmarshal", func(t *testing.T) {
		source := `{"target":"dummy:failure#hello-world", "status":"FAILURE", "message":"it's incident", "starts_at":"2021-01-02T15:04:05Z"}`
		expect := ayd.Incident{
			Target:   &ayd.URL{Scheme: "dummy", Opaque: "failure", Fragment: "hello-world"},
			Status:   ayd.StatusFailure,
			Message:  "it's incident",
			StartsAt: time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		}

		var i ayd.Incident
		if err := json.Unmarshal([]byte(source), &i); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		assert(t, expect, i)
	})
}
