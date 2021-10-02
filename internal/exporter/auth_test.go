package exporter_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/macrat/ayd/internal/exporter"
)

type TestHandler struct{}

func (h TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func TestNewBasicAuth(t *testing.T) {
	tests := []struct {
		Input              string
		IsNeedAuth         bool
		Username, Password string
	}{
		{"", false, "", ""},
		{"hello", true, "hello", ""},
		{"foo:bar", true, "foo", "bar"},
		{"foo:", true, "foo", ""},
		{":bar", true, "", "bar"},
		{"abc:def:ghi", true, "abc", "def:ghi"},
	}

	for _, tt := range tests {
		th := TestHandler{}
		h := exporter.NewBasicAuth(th, tt.Input)

		if !tt.IsNeedAuth {
			if h != th {
				t.Errorf("%#v: expected don't wrap handler but wrapped", tt.Input)
			}
		} else {
			a, ok := h.(exporter.BasicAuth)
			if !ok {
				t.Errorf("%#v: expected wrap with exporter.BasicAuth but not wrapped", tt.Input)
				continue
			}

			if a.Username != tt.Username {
				t.Errorf("%#v: expected username %#v but got %#v", tt.Input, tt.Username, a.Username)
			}

			if a.Password != tt.Password {
				t.Errorf("%#v: expected password %#v but got %#v", tt.Input, tt.Password, a.Password)
			}
		}
	}
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		Userinfo string
		URL      string
		Code     int
	}{
		{"", "http://localhost", http.StatusOK},
		{"foo:bar", "http://foo:bar@localhost", http.StatusOK},
		{"foo:bar", "http://invalid-user:bar@localhost", http.StatusUnauthorized},
		{"foo:bar", "http://foo:invalid-password@localhost", http.StatusUnauthorized},
		{"foo:", "http://foo@localhost", http.StatusOK},
		{"foo:", "http://foo@localhost", http.StatusOK},
		{"foo:", "http://invalid-user@localhost", http.StatusUnauthorized},
		{":bar", "http://:bar@localhost", http.StatusOK},
		{":bar", "http://:invalid-password@localhost", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.URL, func(t *testing.T) {
			h := exporter.NewBasicAuth(TestHandler{}, tt.Userinfo)
			server := httptest.NewServer(h)
			defer server.Close()

			tt.URL = strings.Replace(tt.URL, "localhost", strings.Replace(server.URL, "http://", "", 1), 1)

			resp, err := server.Client().Get(tt.URL)
			if err != nil {
				t.Fatalf("failed to fetch: %s", err)
			}

			if resp.StatusCode != tt.Code {
				t.Fatalf("expected status code %d but got %d", tt.Code, resp.StatusCode)
			}
		})
	}
}
