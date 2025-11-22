package endpoint_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/endpoint"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestHealthzEndpoint(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("failed to get /healthz: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %s", err)
	}

	if string(body) != "HEALTHY\n" {
		t.Errorf("unexpected response:\n%s", body)
	}
}

type DummyErrorsGetter struct {
	healthy  bool
	messages []string
}

func (d DummyErrorsGetter) Name() string {
	return "dummy"
}

func (d DummyErrorsGetter) Path() string {
	return ""
}

func (d DummyErrorsGetter) ProbeHistory() []api.ProbeHistory {
	return nil
}

func (d DummyErrorsGetter) Targets() []string {
	return nil
}

func (d DummyErrorsGetter) MakeReport(length int) api.Report {
	return api.Report{}
}

func (d DummyErrorsGetter) ReportInternalError(_, _ string) {
}

func (d DummyErrorsGetter) Errors() (healthy bool, messages []string) {
	return d.healthy, d.messages
}

func (d DummyErrorsGetter) IncidentCount() int {
	return 0
}

func (d DummyErrorsGetter) String() string {
	return fmt.Sprintf("healthy:%v/messages:%v", d.healthy, d.messages)
}

func (d DummyErrorsGetter) OpenLog(since, until time.Time) (api.LogScanner, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestHealthzEndpoint_errors(t *testing.T) {
	tests := []struct {
		Store DummyErrorsGetter
		Code  int
		Body  string
	}{
		{DummyErrorsGetter{true, []string{}}, http.StatusOK, "HEALTHY\n"},
		{DummyErrorsGetter{true, []string{"hello", "world"}}, http.StatusOK, "HEALTHY\nhello\nworld\n"},
		{DummyErrorsGetter{false, []string{}}, http.StatusInternalServerError, "FAILURE\n"},
		{DummyErrorsGetter{false, []string{"hello", "world"}}, http.StatusInternalServerError, "FAILURE\nhello\nworld\n"},
	}

	for _, tt := range tests {
		t.Run(tt.Store.String(), func(t *testing.T) {
			fun := endpoint.HealthzEndpoint(tt.Store)

			w := httptest.NewRecorder()
			r, err := http.NewRequest("GET", "http://localhost/healthz", nil)
			if err != nil {
				t.Fatalf("failed to prepare http request: %s", err)
			}

			fun(w, r)

			if w.Code != tt.Code {
				t.Errorf("expected status code is %d but got %d", tt.Code, w.Code)
			}

			if w.Body.String() != tt.Body {
				t.Errorf("expected:\n%s\nbut got:\n%s", tt.Body, w.Body)
			}
		})
	}
}
