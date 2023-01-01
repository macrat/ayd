package ayd_test

import (
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
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

		if len(ph1.Records) != len(ph2.Records) {
			t.Errorf("the length of Records is different: %#v != %#v", ph1.Records, ph2.Records)
		} else {
			for i := range ph1.Records {
				if ph1.Records[i].String() != ph2.Records[i].String() {
					t.Errorf("Records[%d] is different: %#v != %#v", i, ph1.Records[i], ph2.Records[i])
				}
			}
		}

		if ph1.Updated != ph2.Updated {
			t.Errorf("the updated is different: %s != %s", ph1.Updated, ph2.Updated)
		}
	}

	ph1 := ayd.ProbeHistory{
		Target: &ayd.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
		Status: ayd.StatusHealthy,
		Records: []ayd.Record{{
			Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
			Status:  ayd.StatusHealthy,
			Latency: 123456 * time.Microsecond,
			Target:  &ayd.URL{Scheme: "dummy", Opaque: "healthy", Fragment: "hello-world"},
			Message: "this is test",
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
		source := `{"target":"dummy:healthy#hello-world", "status":"HEALTHY", "records":[{"time":"2021-01-02T15:04:05Z", "status":"HEALTHY", "latency":123.456, "target":"dummy:healthy#hello-world", "message":"this is test"}], "updated":"2001-01-02T15:04:05Z"}`

		var ph2 ayd.ProbeHistory
		if err := json.Unmarshal([]byte(source), &ph2); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		assert(t, ph1, ph2)
	})
}

func TestSortProbeHistories(t *testing.T) {
	xs := []ayd.ProbeHistory{
		{
			&ayd.URL{Scheme: "a", Opaque: "1"},
			ayd.StatusUnknown,
			time.Now(),
			[]ayd.Record{},
		},
		{
			&ayd.URL{Scheme: "a", Opaque: "2"},
			ayd.StatusHealthy,
			time.Now(),
			[]ayd.Record{{Status: ayd.StatusHealthy}},
		},
		{
			&ayd.URL{Scheme: "a", Opaque: "3"},
			ayd.StatusHealthy,
			time.Now(),
			[]ayd.Record{{Status: ayd.StatusHealthy}},
		},
		{
			&ayd.URL{Scheme: "a", Opaque: "4"},
			ayd.StatusFailure,
			time.Now(),
			[]ayd.Record{{Status: ayd.StatusFailure}},
		},
		{
			&ayd.URL{Scheme: "b", Opaque: "1"},
			ayd.StatusUnknown,
			time.Now(),
			[]ayd.Record{{Status: ayd.StatusUnknown}},
		},
		{
			&ayd.URL{Scheme: "b", Opaque: "2"},
			ayd.StatusAborted,
			time.Now(),
			[]ayd.Record{{Status: ayd.StatusAborted}},
		},
	}

	ayd.SortProbeHistories(xs)

	var ss []string
	for _, x := range xs {
		ss = append(ss, x.Target.String())
	}

	s := strings.Join(ss, ",")

	want := "a:4,b:1,b:2,a:2,a:3,a:1"
	if s != want {
		t.Errorf("unexpected sorted result:\nexpected: %s\n but got: %s", want, s)
	}
}
