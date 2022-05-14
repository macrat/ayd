package ayd_test

import (
	"encoding/json"
	"testing"
	"time"

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

		if i1.CausedAt.String() != i2.CausedAt.String() {
			t.Errorf("the caused_at is different: %s != %s", i1.CausedAt, i2.CausedAt)
		}

		if i1.ResolvedAt.String() != i2.ResolvedAt.String() {
			t.Errorf("the resolved_at is different: %s != %s", i1.ResolvedAt, i2.ResolvedAt)
		}
	}

	t.Run("marshal-and-unmarshal", func(t *testing.T) {
		i1 := ayd.Incident{
			Target:     &ayd.URL{Scheme: "dummy", Opaque: "failure", Fragment: "hello-world"},
			Status:     ayd.StatusFailure,
			Message:    "it's incident",
			CausedAt:   time.Date(2001, 1, 2, 15, 4, 5, 0, time.UTC),
			ResolvedAt: time.Date(2021, 6, 5, 16, 3, 2, 0, time.UTC),
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
		source := `{"target":"dummy:failure#hello-world", "status":"FAILURE", "message":"it's incident", "caused_at":"2021-01-02T15:04:05Z"}`
		expect := ayd.Incident{
			Target:   &ayd.URL{Scheme: "dummy", Opaque: "failure", Fragment: "hello-world"},
			Status:   ayd.StatusFailure,
			Message:  "it's incident",
			CausedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		}

		var i ayd.Incident
		if err := json.Unmarshal([]byte(source), &i); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		assert(t, expect, i)
	})
}
