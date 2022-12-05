package endpoint_test

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/store"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestLogScanner(t *testing.T) {
	tests := []struct {
		Name   string
		Output []string
		Since  time.Time
		Until  time.Time
	}{
		{
			"read_all",
			[]string{"first", "second", "last"},
			time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"drop_first",
			[]string{"second", "last"},
			time.Date(2000, 1, 1, 13, 2, 4, 0, time.UTC),
			time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"drop_last",
			[]string{"first", "second"},
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
		},
		{
			"drop_both",
			[]string{"second"},
			time.Date(2000, 1, 1, 13, 2, 4, 0, time.UTC),
			time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
		},
		{
			"no_match",
			[]string{},
			time.Date(2000, 2, 1, 13, 2, 3, 0, time.UTC),
			time.Date(2000, 2, 1, 13, 2, 3, 0, time.UTC),
		},
		{
			"reverse",
			[]string{},
			time.Date(2000, 2, 1, 13, 2, 3, 0, time.UTC),
			time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
		},
	}

	scanners := []struct {
		Name string
		F    func(since, until time.Time) api.LogScanner
	}{
		{
			"api.NewLogScannerWithPeriod",
			func(since, until time.Time) api.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					`{"time":"2000-01-01T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy","message":"first"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy","message":"second"}`,
					`{"time":"2000-01-03T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy","message":"last"}`,
				}, "\n")))

				return api.NewLogScannerWithPeriod(f, since, until)
			},
		},
		{
			"LogGenerator",
			func(since, until time.Time) api.LogScanner {
				s, err := store.New("", io.Discard)
				if err != nil {
					t.Fatalf("failed to create store: %s", err)
				}

				s.Report(&api.URL{Scheme: "dummy"}, api.Record{
					Time:    time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
					Target:  &api.URL{Scheme: "dummy", Fragment: "hello"},
					Message: "first",
				})
				s.Report(&api.URL{Scheme: "dummy"}, api.Record{
					Time:    time.Date(2000, 1, 2, 13, 2, 3, 0, time.UTC),
					Target:  &api.URL{Scheme: "dummy", Fragment: "world"},
					Message: "second",
				})
				s.Report(&api.URL{Scheme: "dummy"}, api.Record{
					Time:    time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
					Target:  &api.URL{Scheme: "dummy", Fragment: "hello"},
					Message: "last",
				})

				scanner, err := s.OpenLog(since, until)
				if err != nil {
					t.Fatalf("failed to create scanner: %s", err)
				}
				return scanner
			},
		},
		{
			"LogFilter-target",
			func(since, until time.Time) api.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					`{"time":"2000-01-01T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#1","message":"first"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#1","message":"second"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"FAILURE","latency":0.123,"target":"dummy:failure","message":"another"}`,
					`{"time":"2000-01-03T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#2","message":"last"}`,
					`{"time":"2000-01-04T13:02:03Z","status":"FAILURE","latency":0.123,"target":"dummy:failure","message":"another"}`,
				}, "\n")))

				return endpoint.LogFilter{
					api.NewLogScannerWithPeriod(f, since, until),
					[]string{"dummy:healthy#1", "dummy:healthy#2"},
					nil,
				}
			},
		},
		{
			"LogFilter-query",
			func(since, until time.Time) api.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					`{"time":"2000-01-01T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#1","message":"first"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#1","message":"second"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"FAILURE","latency":0.123,"target":"dummy:failure","message":"another"}`,
					`{"time":"2000-01-03T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#2","message":"last"}`,
					`{"time":"2000-01-04T13:02:03Z","status":"FAILURE","latency":0.123,"target":"dummy:failure","message":"another"}`,
				}, "\n")))

				return endpoint.LogFilter{
					api.NewLogScannerWithPeriod(f, since, until),
					nil,
					endpoint.ParseQuery("healthy"),
				}
			},
		},
	}

	for _, scanner := range scanners {
		t.Run(scanner.Name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.Name, func(t *testing.T) {
					r := scanner.F(tt.Since, tt.Until)
					defer r.Close()

					var results []string
					for r.Scan() {
						results = append(results, r.Record().Message)
					}

					if len(results) != len(tt.Output) {
						t.Fatalf("unexpected number of output: %d\n%#v", len(results), results)
					}

					for i := range results {
						if results[i] != tt.Output[i] {
							t.Errorf("unexpected message at %d: %#v != %#v", i, results[i], tt.Output[i])
						}
					}
				})
			}
		})
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		Query  string
		Record api.Record
		Expect bool
	}{
		{
			"dummy:healthy",
			api.Record{
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"dummy:",
			api.Record{
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"foo bar",
			api.Record{
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"healthy bar",
			api.Record{
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"healthy baz",
			api.Record{
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			"failure bar",
			api.Record{
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			"failure healthy",
			api.Record{
				Status:  api.StatusFailure,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"-unknown",
			api.Record{
				Status:  api.StatusFailure,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"-HEALTHY",
			api.Record{
				Status:  api.StatusFailure,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			"<100ms",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"<10ms >0s",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			">=50ms <=1s",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"=50ms !=100ms",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &api.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Query, func(t *testing.T) {
			result := endpoint.ParseQuery(tt.Query).Match(tt.Record)
			if result != tt.Expect {
				t.Errorf("expected %#v but got %#v", tt.Expect, result)
			}
		})
	}
}

func TestLogJsonEndpoint(t *testing.T) {
	tests := []struct {
		Name       string
		Query      string
		StatusCode int
		Length     int
		Error      string
	}{
		{
			"without-query",
			"",
			http.StatusOK,
			0,
			"",
		},
		{
			"fetch-all",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			3,
			"",
		},
		{
			"drop-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://b.example.com",
			http.StatusOK,
			2,
			"",
		},
		{
			"invalid-since",
			"?since=invalid-since&until=2022-01-01T00:00:00Z",
			http.StatusBadRequest,
			0,
			"invalid query format: since",
		},
		{
			"invalid-until",
			"?since=2021-01-01T00:00:00Z&until=invalid-until",
			http.StatusBadRequest,
			0,
			"invalid query format: until",
		},
		{
			"invalid-since-and-until",
			"?since=invalid-since&until=invalid-until",
			http.StatusBadRequest,
			0,
			"invalid query format: until, since",
		},
	}

	srv := testutil.StartTestServer(t)
	t.Cleanup(func() {
		srv.Close()
	})

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			resp, err := srv.Client().Get(srv.URL + "/log.json" + tt.Query)
			if err != nil {
				t.Fatalf("failed to get /log.json: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.StatusCode {
				t.Errorf("unexpected status: %s", resp.Status)
			}

			dec := json.NewDecoder(resp.Body)

			if tt.Error == "" {
				var result struct {
					Records []api.Record `json:"records"`
				}

				if err = dec.Decode(&result); err != nil {
					t.Fatalf("failed to read result: %s", err)
				}

				if len(result.Records) != tt.Length {
					t.Errorf("unexpected count of result: %#v", result)
				}
			} else {
				var result struct {
					Error string `json:"error"`
				}

				if err = dec.Decode(&result); err != nil {
					t.Fatalf("failed to read result: %s", err)
				}

				if result.Error != tt.Error {
					t.Errorf("unexpected error message: %#v", result.Error)
				}
			}
		})
	}
}

func TestLogCSVEndpoint(t *testing.T) {
	tests := []struct {
		Name       string
		Query      string
		StatusCode int
		Pattern    string
	}{
		{
			"without-query",
			"",
			http.StatusOK,
			"time,status,latency,target,message,extra\n",
		},
		{
			"fetch-all",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n(.*\n){3}",
		},
		{
			"drop-with-time-range",
			"?since=2021-01-02T15:04:06Z&until=2021-01-02T15:04:07Z&target=http://a.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n2021-01-02T15:04:06Z,.*\n",
		},
		{
			"drop-all-with-time-range",
			"?since=2001-01-01T00:00:00Z&until=2002-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n",
		},
		{
			"drop-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://b.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n(.*\n){2}",
		},
		{
			"drop-all-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://no-such.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n",
		},
		{
			"invalid-since",
			"?since=invalid-since&until=2022-01-01T00:00:00Z",
			http.StatusBadRequest,
			"invalid query format: since\n",
		},
		{
			"invalid-until",
			"?since=2021-01-01T00:00:00Z&until=invalid-until",
			http.StatusBadRequest,
			"invalid query format: until\n",
		},
		{
			"invalid-since-and-until",
			"?since=invalid-since&until=invalid-until",
			http.StatusBadRequest,
			"invalid query format: until, since\n",
		},
	}

	srv := testutil.StartTestServer(t)
	t.Cleanup(func() {
		srv.Close()
	})

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			resp, err := srv.Client().Get(srv.URL + "/log.csv" + tt.Query)
			if err != nil {
				t.Fatalf("failed to get /log.csv: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.StatusCode {
				t.Errorf("unexpected status: %s", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}

			if ok, err := regexp.Match("^"+tt.Pattern+"$", body); err != nil {
				t.Errorf("failed to check body: %s", err)
			} else if !ok {
				t.Errorf("body must match to %#v but got:\n%s", tt.Pattern, string(body))
			}
		})
	}
}

func TestLogLTSVEndpoint(t *testing.T) {
	tests := []struct {
		Name       string
		Query      string
		StatusCode int
		Pattern    string
	}{
		{
			"without-query",
			"",
			http.StatusOK,
			"",
		},
		{
			"fetch-all",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			"(time:[^\t]*\tstatus:[^\t]{7}\tlatency:[^\t]*\ttarget:[^\t]*\tmessage:[^\t]*\n){3}",
		},
		{
			"drop-with-time-range",
			"?since=2021-01-02T15:04:06Z&until=2021-01-02T15:04:07Z&target=http://a.example.com",
			http.StatusOK,
			"time:2021-01-02T15:04:06Z\tstatus:[^\t]{7}\tlatency:[^\t]*\ttarget:http://a\\.example\\.com*\tmessage:[^\t]*\n",
		},
		{
			"drop-all-with-time-range",
			"?since=2001-01-01T00:00:00Z&until=2002-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			"",
		},
		{
			"drop-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://b.example.com",
			http.StatusOK,
			"(time:[^\t]*\tstatus:[^\t]{7}\tlatency:[^\t]*\ttarget:http://b\\.example\\.com\tmessage:[^\t]*\n){2}",
		},
		{
			"drop-all-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://no-such.example.com",
			http.StatusOK,
			"",
		},
		{
			"invalid-since",
			"?since=invalid-since&until=2022-01-01T00:00:00Z",
			http.StatusBadRequest,
			"invalid query format: since\n",
		},
		{
			"invalid-until",
			"?since=2021-01-01T00:00:00Z&until=invalid-until",
			http.StatusBadRequest,
			"invalid query format: until\n",
		},
		{
			"invalid-since-and-until",
			"?since=invalid-since&until=invalid-until",
			http.StatusBadRequest,
			"invalid query format: until, since\n",
		},
	}

	srv := testutil.StartTestServer(t)
	t.Cleanup(func() {
		srv.Close()
	})

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			resp, err := srv.Client().Get(srv.URL + "/log.ltsv" + tt.Query)
			if err != nil {
				t.Fatalf("failed to get /log.ltsv: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.StatusCode {
				t.Errorf("unexpected status: %s", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %s", err)
			}

			if ok, err := regexp.Match("^"+tt.Pattern+"$", body); err != nil {
				t.Errorf("failed to check body: %s", err)
			} else if !ok {
				t.Errorf("body must match to %#v but got:\n%s", tt.Pattern, string(body))
			}
		})
	}
}

func TestLogHTMLEndpoint(t *testing.T) {
	AssertEndpoint(t, "/log.html?since=2021-01-01T00%3A00%3A00Z&until=2021-01-03T00%3A00%3A00Z", "./testdata/log.html", "[0-9] years? ago")
}
