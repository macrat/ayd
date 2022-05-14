package main_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/macrat/ayd"
	"github.com/macrat/ayd/internal/ayderr"
	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

type PanicProbe struct{}

func (p PanicProbe) Target() *api.URL {
	return &api.URL{Scheme: "test", Opaque: "panic"}
}

func (p PanicProbe) Probe(ctx context.Context, r scheme.Reporter) {
	panic("this always make panic")
}

func TestMakeJob(t *testing.T) {
	s := testutil.NewStore(t)
	defer s.Close()

	task := main.Task{Prober: PanicProbe{}}
	task.MakeJob(context.Background(), s).Run()

	history := s.ProbeHistory()[0]

	if len(history.Records) != 1 {
		t.Fatalf("unexpected length history found\nhistory.Records = %#v", history.Records)
	}

	r := history.Records[0]

	if r.Status != api.StatusUnknown {
		t.Errorf("unexpected status: %s", r.Status)
	}
	if r.Message != "panic: this always make panic" {
		t.Errorf("unexpected message: %s", r.Message)
	}
}

func TestParseArgs(t *testing.T) {
	type WantTask struct {
		Schedule string
		Target   string
	}

	tests := []struct {
		Args []string
		Want []WantTask
	}{
		{
			Args: []string{"ping:hoge", "2m", "http://example.com"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"2m0s", "http://example.com/"},
			},
		},
		{
			Args: []string{"ping:hoge", "ping://fuga", "2m", "1h", "http://example.com"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"5m0s", "ping:fuga"},
				{"1h0m0s", "http://example.com/"},
			},
		},
		{
			Args: []string{"ping:hoge", "2m", "ping:hoge", "5m", "ping:hoge?abc", "2m", "ping://hoge"},
			Want: []WantTask{
				{"5m0s", "ping:hoge"},
				{"2m0s", "ping:hoge"},
			},
		},
		{
			Args: []string{"*/10  * * *", "ping:hoge", "*/5 *\t* * 0", "ping:fuga"},
			Want: []WantTask{
				{"*/10 * * * ?", "ping:hoge"},
				{"*/5 * * * 0", "ping:fuga"},
			},
		},
	}

	for _, tt := range tests {
		result, err := main.ParseArgs(tt.Args)
		if err != nil {
			t.Errorf("%#v: failed parse:\n%s", tt.Args, err)
			continue
		}

		if len(result) != len(tt.Want) {
			t.Errorf("%#v: expected %d tasks but got %d tasks", tt.Args, len(tt.Want), len(result))
			continue
		}

		for i := range result {
			if result[i].Prober.Target().String() != tt.Want[i].Target {
				t.Errorf("%#v: unexpected target at %d: expected %#v but got %#v", tt.Args, i, tt.Want[i].Target, result[i].Prober.Target().String())
			}

			if result[i].Schedule.String() != tt.Want[i].Schedule {
				t.Errorf("%#v: unexpected interval at %d: expected %s but got %s", tt.Args, i, tt.Want[i].Schedule, result[i].Schedule)
			}
		}
	}
}

func TestParseArgs_errors(t *testing.T) {
	tests := []struct {
		Args   []string
		Errors []string
	}{
		{
			[]string{"* * * *", "ping:localhost"},
			[]string{},
		},
		{
			[]string{"no.such.scheme:hello-world", "hello-world"},
			[]string{
				`no.such.scheme:hello-world: This scheme is not supported. Please check if the plugin is installed if need.`,
				`hello-world: Not valid as schedule or target URL. Please specify scheme if this is target. (e.g. ping:hello-world or http://hello-world)`,
			},
		},
		{
			[]string{"http-abc://example.com", "::"},
			[]string{
				`http-abc://example.com: HTTP "ABC" method is not supported. Please use GET, HEAD, POST, OPTIONS, or CONNECT.`,
				`::: Not valid as schedule or target URL.`,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(strings.Join(tt.Args, ","), func(t *testing.T) {
			_, err := main.ParseArgs(tt.Args)

			if len(tt.Errors) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}

			es := ayderr.List{}
			if !errors.As(err, &es) {
				t.Fatalf("unexpected error: %#v", err)
			}

			if len(es.Children) != len(tt.Errors) {
				t.Fatalf("unexpected count of errors: expected %d but got %d", len(tt.Errors), len(es.Children))
			}

			for i, e := range es.Children {
				if e.Error() != tt.Errors[i] {
					t.Errorf("%d: unexpected error message\nexpected: %s\n but got: %s", i, tt.Errors[i], e)
				}
			}
		})
	}
}
