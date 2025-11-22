package endpoint_test

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/query"
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
				s, err := store.New("", "", io.Discard)
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
			"FilterScanner",
			func(since, until time.Time) api.LogScanner {
				f := io.NopCloser(strings.NewReader(strings.Join([]string{
					`{"time":"2000-01-01T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#1","message":"first"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#1","message":"second"}`,
					`{"time":"2000-01-02T13:02:03Z","status":"FAILURE","latency":0.123,"target":"dummy:failure","message":"another"}`,
					`{"time":"2000-01-03T13:02:03Z","status":"HEALTHY","latency":0.123,"target":"dummy:healthy#2","message":"last"}`,
					`{"time":"2000-01-04T13:02:03Z","status":"FAILURE","latency":0.123,"target":"dummy:failure","message":"another"}`,
				}, "\n")))

				return endpoint.FilterScanner{
					api.NewLogScannerWithPeriod(f, since, until),
					query.ParseQuery("healthy"),
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

func TestPagingScanner(t *testing.T) {
	s := testutil.NewStoreWithLog(t)

	messages := []string{
		"hello world",
		"this is failure",
		"hello world!",
		"this is healthy",
		"hello world!!",
		"this is aborted",
		"this is unknown",
	}

	tests := []struct {
		Offset  uint64
		Limit   uint64
		Records []string
	}{
		{0, 0, messages},
		{0, 100, messages},
		{1, 0, messages[1:]},
		{5, 0, messages[5:]},
		{0, uint64(len(messages)), messages},
		{0, uint64(len(messages) - 1), messages[:len(messages)-1]},
		{2, 3, messages[2:5]},
		{1, 1, messages[1:2]},
		{2, 1, messages[2:3]},
		{0, 2, messages[:2]},
		{0, 1, messages[:1]},
		{5, 10, messages[5:]},
	}

	for _, tt := range tests {
		r, err := s.OpenLog(time.Unix(0, 0), time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("failed to open log: %s", err)
		}
		defer r.Close()

		ps := endpoint.PagingScanner{
			Scanner: r,
			Offset:  tt.Offset,
			Limit:   tt.Limit,
		}

		var records []string
		for ps.Scan() {
			records = append(records, ps.Record().Message)
		}
		total := ps.ScanTotal()

		if diff := cmp.Diff(tt.Records, records); diff != "" {
			t.Errorf("%d-%d: unexpected records:\n%s", tt.Offset, tt.Limit, diff)
		}

		if total != uint64(len(messages)) {
			t.Errorf("%d-%d: expected total is %d but got %d", tt.Offset, tt.Limit, len(messages), total)
		}
	}
}

func TestContextScanner_scanAll(t *testing.T) {
	s := testutil.NewStoreWithLog(t)

	r, err := s.OpenLog(time.Unix(0, 0), time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to open log: %s", err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cs := endpoint.NewContextScanner(ctx, r)

	count := 0
	for cs.Scan() {
		t.Log(cs.Record())
		count++
	}

	if count != 7 {
		t.Fatalf("unexpected number of records found: %d", count)
	}
}

func TestContextScanner_cancel(t *testing.T) {
	s := testutil.NewStoreWithLog(t)

	r, err := s.OpenLog(time.Unix(0, 0), time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to open log: %s", err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())

	cs := endpoint.NewContextScanner(ctx, r)

	if !cs.Scan() {
		cancel()
		t.Fatalf("failed to scan first record")
	}

	cancel()

	if cs.Scan() {
		t.Fatalf("unexpectedly succeed to scan record: %s", cs.Record())
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
			"?q=time%3E%3D2021-01-01+time%3C2022-01-01",
			http.StatusOK,
			7,
			"",
		},
		{
			"drop-with-timerange",
			"?q=time%3E%3D2021-01-02T15:04:06Z+time%3C2021-01-02T15:04:08Z",
			http.StatusOK,
			3,
			"",
		},
		{
			"drop-with-target",
			"?q=time%3E%3D2021-01-01+time%3C2022-01-01+target%3Dhttp://b.example.com",
			http.StatusOK,
			2,
			"",
		},
		{
			"with-limit",
			"?q=time%3E%3D2021-01-01+time%3C2022-01-01&limit=3",
			http.StatusOK,
			3,
			"",
		},
		{
			"with-offset-and-limit",
			"?q=time%3E%3D2021-01-01+time%3C2022-01-01&limit=3&offset=5",
			http.StatusOK,
			2, // limit is 3 but only 2 records exist.
			"",
		},
		{
			"invalid-limit",
			"?limit=abc",
			http.StatusBadRequest,
			0,
			"invalid query format: limit",
		},
		{
			"invalid-offset",
			"?offset=01fa",
			http.StatusBadRequest,
			0,
			"invalid query format: offset",
		},
		{
			"invalid-limit-and-offset",
			"?offset=1a&limit=3f",
			http.StatusBadRequest,
			0,
			"invalid query format: limit, offset",
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
				t.Fatalf("unexpected status: %s", resp.Status)
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
					t.Errorf("unexpected count of result: %d", len(result.Records))
					for _, r := range result.Records {
						t.Log(r)
					}
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

	t.Run("snapshot", func(t *testing.T) {
		AssertEndpoint(t, "/log.json?q=time%3E%3D20210102T150406Z+time%3C20210102T150409Z&limit=2&offset=1", "./testdata/log.json", "[0-9] years? ago")
	})
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
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z",
			http.StatusOK,
			"time,status,latency,target,message,extra\n(.*\n){7}",
		},
		{
			"drop-with-time-range",
			"?q=time%3E%3D2021-01-02T15:04:06Z+time%3C2021-01-02T15:04:07Z+target%3Dhttp://a.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n2021-01-02T15:04:06Z,.*\n",
		},
		{
			"drop-all-with-time-range",
			"?q=time%3E%3D2001-01-01T00:00:00Z+time%3C2002-01-01T00:00:00Z+target%3Dhttp://a.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n",
		},
		{
			"drop-with-target",
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z+target%3Dhttp://b.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n(.*\n){2}",
		},
		{
			"drop-all-with-target",
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z+target%3Dhttp://no-such.example.com",
			http.StatusOK,
			"time,status,latency,target,message,extra\n",
		},
		{
			"with-offset-and-limit",
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z&offset=1&limit=2",
			http.StatusOK,
			"time,status,latency,target,message,extra\n2021-01-02T15:04:05Z,[^\n]*\n2021-01-02T15:04:06Z,[^\n]*\n",
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

func TestLogXlsxEndpoint(t *testing.T) {
	srv := testutil.StartTestServer(t)
	t.Cleanup(func() {
		srv.Close()
	})

	resp, err := srv.Client().Get(srv.URL + "/log.xlsx")
	if err != nil {
		t.Fatalf("failed to get /log.xlsx: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("unexpected status: %s", resp.Status)
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
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z",
			http.StatusOK,
			"(time:[^\t]*\tstatus:[^\t]{7}\tlatency:[^\t]*\ttarget:[^\t]*\tmessage:[^\t]*(\t[a-z]+:[^\t]*)*\n){7}",
		},
		{
			"drop-with-time-range",
			"?q=time%3E%3D2021-01-02T15:04:06Z+time%3C2021-01-02T15:04:07Z+target%3Dhttp://a.example.com",
			http.StatusOK,
			"time:2021-01-02T15:04:06Z\tstatus:[^\t]{7}\tlatency:[^\t]*\ttarget:http://a\\.example\\.com*\tmessage:[^\t]*\n",
		},
		{
			"drop-all-with-time-range",
			"?q=time%3E%3D2001-01-01T00:00:00Z+time%3C2002-01-01T00:00:00Z+target%3Dhttp://a.example.com",
			http.StatusOK,
			"",
		},
		{
			"drop-with-target",
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z+target%3Dhttp://b.example.com",
			http.StatusOK,
			"(time:[^\t]*\tstatus:[^\t]{7}\tlatency:[^\t]*\ttarget:http://b\\.example\\.com\tmessage:[^\t]*(\textra:[^\t]*)?\n){2}",
		},
		{
			"drop-all-with-target",
			"?q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z+target%3Dhttp://no-such.example.com",
			http.StatusOK,
			"",
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
	AssertEndpoint(t, "/log.html?q=time%3E%3D2021-01-01T00%3A00%3A00Z+time%3C2021-01-03T00%3A00%3A00Z", "./testdata/log.html", `[0-9] years? ago|time\&gt;=20\d{2}-\d{2}-\d{2}T\d{2}:\d{2}`)
}
