package ayd_test

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/macrat/ayd/lib-ayd"
)

func TestProbeHistory(t *testing.T) {
	assert := func(t *testing.T, ph1, ph2 ayd.ProbeHistory) {
		if ph1.Target.String() != ph2.Target.String() {
			t.Errorf("the target is different: %s != %s", ph1.Target, ph2.Target)
		}

		if ph1.Status != ph2.Status {
			t.Errorf("the status is different: %s != %s", ph1.Status, ph2.Status)
		}

		if len(ph1.History) != len(ph2.History) {
			t.Errorf("the length of History is different: %#v != %#v", ph1.History, ph2.History)
		} else {
			for i := range ph1.History {
				if ph1.History[i].String() != ph2.History[i].String() {
					t.Errorf("History[%d] is different: %#v != %#v", i, ph1.History[i], ph2.History[i])
				}
			}
		}

		if ph1.Updated != ph2.Updated {
			t.Errorf("the updated is different: %s != %s", ph1.Updated, ph2.Updated)
		}
	}

	ph1 := ayd.ProbeHistory{
		Target: &url.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
		Status: ayd.StatusHealthy,
		History: []ayd.Record{{
			CheckedAt: time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
			Status:    ayd.StatusHealthy,
			Latency:   123456 * time.Microsecond,
			Target:    &url.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
			Message:   "this is test",
		}},
		Updated: time.Date(2001, 1, 2, 15, 4, 5, 0, time.UTC),
	}

	t.Run("marshal-and-unmarshal", func(t *testing.T) {
		j, err := json.Marshal(ph1)
		if err != nil {
			t.Fatalf("failed to marshal: %s", err)
		}

		t.Log(string(j))

		var ph2 ayd.ProbeHistory
		err = json.Unmarshal(j, &ph2)
		if err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		assert(t, ph1, ph2)
	})

	t.Run("unmarshal", func(t *testing.T) {
		source := `{"target":"dummy:healthy#hello-world", "status":"HEALTHY", "history":[{"checked_at":"2021-01-02T15:04:05Z", "status":"HEALTHY", "latency":123.456, "target":"dummy:healthy#hello-world", "message":"this is test"}], "updated":"2001-01-02T15:04:05Z"}`

		var ph2 ayd.ProbeHistory
		if err := json.Unmarshal([]byte(source), &ph2); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		assert(t, ph1, ph2)
	})
}
