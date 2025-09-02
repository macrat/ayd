package endpoint_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func FuzzLogJsonEndpoint(f *testing.F) {
	s := testutil.NewStoreWithLog(f)
	defer s.Close()
	handler := endpoint.LogJsonEndpoint(s)

	f.Add("q=time%3C2021-01-02T15:04:05+09:00")
	f.Add("q=time%3E1999-12-31T23:59:59-12:00")
	f.Add("q=time%3D123456789")
	f.Add("q=time%3D981183906")
	f.Add("q=time%3E%3D2001-02-01T01:23:45Z+time%3C2123-10-09T20:07:06+01:00")
	f.Add("q=time%3Dhttp://localhost")
	f.Add("q=time%3E%3D2001-02-01T01:23:45Z+time%3C2123-10-09T20:07:06+01:00+target%3Dhttp://localhost")
	f.Add("q=time%3E%3D2021-01-01T00:00:00Z+time%3C2022-01-01T00:00:00Z+target%3Dhttp://a.example.com")
	f.Add("q=time%3E%3D2021-01-02T15:04:06Z+time%3C2021-01-02T15:04:07Z+target%3Dhttp://a.example.com")
	f.Add("q=time%3E%3D2001-01-01T00:00:00Z+time%3C2002-01-01T00:00:00Z+target%3Dhttp://a.example.com")
	f.Add("q=time%3E%3D981183906+time%3C2022-01-01T00:00:00Z+target%3Dhttp://b.example.com")
	f.Add("q=time%3E%3Dinvalid-since+time%3C%3D2022-01-01T00:00:00Z")
	f.Add("q=time%3E%3D2021-01-01T00:00:00Z+time%3Cinvalid-until")
	f.Add("q=%3c10ms")
	f.Add("q=%3e10ms%20%3c20ms")
	f.Add("q=http://a.example.com+OR+http://b.example.com+%3e%3d100ms")
	f.Add("q=http://a.example.com+healthy")

	f.Fuzz(func(t *testing.T, query string) {
		query = strings.ReplaceAll(query, ";", "%3B")

		req, err := http.NewRequest("GET", "http://localhost:9000/log.json?"+query, nil)
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
			t.Fatalf("unexpected status code from /log.json?%s: %d", query, resp.Code)
		}

		body := resp.Body.String()

		if body == "" {
			return
		}
		if body[len(body)-1] != '\n' {
			t.Fatalf("the last character should be \\n but got %q", body[len(body)-1])
		}
		body = body[:len(body)-1]

		var response struct {
			R []api.Record `json:"records"`
		}

		if err := json.Unmarshal([]byte(body), &response); err != nil {
			t.Errorf("failed to parse response: %d", err)
		}
	})
}
