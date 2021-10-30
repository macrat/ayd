//go:build gofuzzbeta
// +build gofuzzbeta

package endpoint_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func FuzzLogTSVEndpoint(f *testing.F) {
	s := testutil.NewStoreWithLog(f)
	defer s.Close()
	handler := endpoint.LogTSVEndpoint(s)

	f.Add("since=2021-01-02T15:04:05+09:00")
	f.Add("until=1999-12-31T23:59:59-12:00")
	f.Add("since=2001-02-01T01:23:45Z&until=2123-10-09T20:07:06+01:00")
	f.Add("target=http://localhost")
	f.Add("since=2001-02-01T01:23:45Z&until=2123-10-09T20:07:06+01:00&target=http://localhost")
	f.Add("since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://a.example.com")
	f.Add("since=2021-01-02T15:04:06Z&until=2021-01-02T15:04:07Z&target=http://a.example.com")
	f.Add("since=2001-01-01T00:00:00Z&until=2002-01-01T00:00:00Z&target=http://a.example.com")
	f.Add("since=2021-01-01T00:00:00Z&until=2022-01-01T00:00:00Z&target=http://b.example.com")
	f.Add("since=invalid-since&until=2022-01-01T00:00:00Z")
	f.Add("since=2021-01-01T00:00:00Z&until=invalid-until")
	f.Add("since=invalid-since&until=invalid-until")

	f.Fuzz(func(t *testing.T, query string) {
		query = strings.ReplaceAll(query, ";", "%3B")

		req, err := http.NewRequest("GET", "http://localhost:9000/log.tsv?"+query, nil)
		if err != nil {
			t.Skip()
		}

		resp := httptest.NewRecorder()
		handler(resp, req)

		if resp.Code == http.StatusBadRequest {
			// The query is incorrect but rejected correctly.
			return
		}
		if resp.Code != http.StatusOK {
			t.Fatalf("unexpected status code from /log.tsv?%s: %d", query, resp.Code)
		}

		body := resp.Body.String()

		if body == "" {
			return
		}
		if body[len(body)-1] != '\n' {
			t.Fatalf("the last character should be \\n but got %q", body[len(body)-1])
		}
		body = body[:len(body)-1]

		for i, line := range strings.Split(body, "\n") {
			_, err := api.ParseRecord(line)
			if err != nil {
				t.Errorf("line %d is incorrect as a log line: %s\n%q", i+1, err, line)
			}
		}
	})
}
