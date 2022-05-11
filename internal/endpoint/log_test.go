package endpoint_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
		F    func(since, until time.Time) endpoint.LogScanner
	}{
		{
			"LogReader",
			func(since, until time.Time) endpoint.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tfirst",
					"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tsecond",
					"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy\tlast",
				}, "\n")))

				return endpoint.NewLogReaderFromReader(f, since, until)
			},
		},
		{
			"LogGenerator",
			func(since, until time.Time) endpoint.LogScanner {
				s, err := store.New("", io.Discard)
				if err != nil {
					t.Fatalf("failed to create store: %s", err)
				}

				s.Report(&url.URL{Scheme: "dummy"}, api.Record{
					CheckedAt: time.Date(2000, 1, 1, 13, 2, 3, 0, time.UTC),
					Target:    &url.URL{Scheme: "dummy", Fragment: "hello"},
					Message:   "first",
				})
				s.Report(&url.URL{Scheme: "dummy"}, api.Record{
					CheckedAt: time.Date(2000, 1, 2, 13, 2, 3, 0, time.UTC),
					Target:    &url.URL{Scheme: "dummy", Fragment: "world"},
					Message:   "second",
				})
				s.Report(&url.URL{Scheme: "dummy"}, api.Record{
					CheckedAt: time.Date(2000, 1, 3, 13, 2, 3, 0, time.UTC),
					Target:    &url.URL{Scheme: "dummy", Fragment: "hello"},
					Message:   "last",
				})

				return endpoint.NewLogGenerator(s, since, until)
			},
		},
		{
			"LogFilter-target",
			func(since, until time.Time) endpoint.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy#1\tfirst",
					"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy#1\tsecond",
					"2000-01-02T13:02:03Z\tFAILURE\t0.123\tdummy:failure\tanother",
					"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy#2\tlast",
					"2000-01-04T13:02:03Z\tFAILURE\t0.123\tdummy:failure\tanother",
				}, "\n")))

				return endpoint.LogFilter{
					endpoint.NewLogReaderFromReader(f, since, until),
					[]string{"dummy:healthy#1", "dummy:healthy#2"},
					nil,
				}
			},
		},
		{
			"LogFilter-query",
			func(since, until time.Time) endpoint.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					"2000-01-01T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy#1\tfirst",
					"2000-01-02T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy#1\tsecond",
					"2000-01-02T13:02:03Z\tFAILURE\t0.123\tdummy:failure\tanother",
					"2000-01-03T13:02:03Z\tHEALTHY\t0.123\tdummy:healthy#2\tlast",
					"2000-01-04T13:02:03Z\tFAILURE\t0.123\tdummy:failure\tanother",
				}, "\n")))

				return endpoint.LogFilter{
					endpoint.NewLogReaderFromReader(f, since, until),
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
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"dummy:",
			api.Record{
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"foo bar",
			api.Record{
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"healthy bar",
			api.Record{
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"healthy baz",
			api.Record{
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			"failure bar",
			api.Record{
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			"failure healthy",
			api.Record{
				Status:  api.StatusFailure,
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"<100ms",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"<10ms >0s",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			false,
		},
		{
			">=50ms <=1s",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
				Message: "foobar",
			},
			true,
		},
		{
			"=50ms !=100ms",
			api.Record{
				Latency: 50 * time.Millisecond,
				Target:  &url.URL{Scheme: "dummy", Opaque: "healthy"},
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

func TestLogTSVEndpoint(t *testing.T) {
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
			"(.*\n){3}",
		},
		{
			"drop-with-time-range",
			"?since=2021-01-02T15:04:06Z&until=2021-01-02T15:04:07Z&target=http://a.example.com",
			http.StatusOK,
			"2021-01-02T15:04:06Z\t.*\n",
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
			"(.*\n){2}",
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

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			srv := testutil.StartTestServer(t)
			defer srv.Close()

			resp, err := srv.Client().Get(srv.URL + "/log.tsv" + tt.Query)
			if err != nil {
				t.Fatalf("failed to get /log.tsv: %s", err)
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

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			srv := testutil.StartTestServer(t)
			defer srv.Close()

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
			"timestamp,status,latency,target,message\n",
		},
		{
			"fetch-all",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			"timestamp,status,latency,target,message\n(.*\n){3}",
		},
		{
			"drop-with-time-range",
			"?since=2021-01-02T15:04:06Z&until=2021-01-02T15:04:07Z&target=http://a.example.com",
			http.StatusOK,
			"timestamp,status,latency,target,message\n2021-01-02T15:04:06Z,.*\n",
		},
		{
			"drop-all-with-time-range",
			"?since=2001-01-01T00:00:00Z&until=2002-01-01T00:00:00Z&target=http://a.example.com",
			http.StatusOK,
			"timestamp,status,latency,target,message\n",
		},
		{
			"drop-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://b.example.com",
			http.StatusOK,
			"timestamp,status,latency,target,message\n(.*\n){2}",
		},
		{
			"drop-all-with-target",
			"?since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://no-such.example.com",
			http.StatusOK,
			"timestamp,status,latency,target,message\n",
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

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			srv := testutil.StartTestServer(t)
			defer srv.Close()

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

func TestLogHTMLEndpoint(t *testing.T) {
	AssertEndpoint(t, "/log.html?since=2021-01-01T00%3A00%3A00Z&until=2021-01-03T00%3A00%3A00Z", "./testdata/log.html", "")
}
