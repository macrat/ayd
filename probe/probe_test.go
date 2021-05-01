package probe_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/macrat/ayd/probe"
	"github.com/macrat/ayd/store"
	"github.com/macrat/ayd/testutil"
)

func TestTargetURLNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Input string
		Want  url.URL
	}{
		{"ping:example.com", url.URL{Scheme: "ping", Opaque: "example.com"}},
		{"ping://example.com:123/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "ping", Opaque: "example.com"}},

		{"http://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"https://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},

		{"http-get://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http-get", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"https-post://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https-post", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"http-head://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http-head", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"https-options://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https-options", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},

		{"tcp:example.com:80", url.URL{Scheme: "tcp", Host: "example.com:80"}},
		{"tcp://example.com:80/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "tcp", Host: "example.com:80"}},

		{"dns:example.com", url.URL{Scheme: "dns", Opaque: "example.com"}},
		{"dns://example.com:80/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "dns", Opaque: "example.com"}},

		{"exec:foo.sh", url.URL{Scheme: "exec", Opaque: "foo.sh"}},
		{"exec:./foo.sh", url.URL{Scheme: "exec", Opaque: "./foo.sh"}},
		{"exec:/foo/bar.sh", url.URL{Scheme: "exec", Opaque: "/foo/bar.sh"}},
		{"exec:///foo/bar.sh", url.URL{Scheme: "exec", Opaque: "/foo/bar.sh"}},
		{"exec:foo.sh?hoge=fuga#piyo", url.URL{Scheme: "exec", Opaque: "foo.sh", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"exec:/foo/bar.sh?hoge=fuga#piyo", url.URL{Scheme: "exec", Opaque: "/foo/bar.sh", RawQuery: "hoge=fuga", Fragment: "piyo"}},

		{"source:./testdata/healthy-list.txt", url.URL{Scheme: "source", Opaque: "./testdata/healthy-list.txt"}},
	}

	for _, tt := range tests {
		p, err := probe.New(tt.Input)
		if err != nil {
			t.Errorf("%#v: failed to parse: %#s", tt.Input, err)
			continue
		}

		u := p.Target()

		if u.Scheme != tt.Want.Scheme {
			t.Errorf("%#v expected scheme %#v but go %#v", tt.Input, tt.Want.Scheme, u.Scheme)
		}

		if u.Opaque != tt.Want.Opaque {
			t.Errorf("%#v expected opaque %#v but go %#v", tt.Input, tt.Want.Opaque, u.Opaque)
		}

		if u.Host != tt.Want.Host {
			t.Errorf("%#v expected host %#v but go %#v", tt.Input, tt.Want.Host, u.Host)
		}

		if u.Path != tt.Want.Path {
			t.Errorf("%#v expected path %#v but go %#v", tt.Input, tt.Want.Path, u.Path)
		}

		if u.Fragment != tt.Want.Fragment {
			t.Errorf("%#v expected fragment %#v but go %#v", tt.Input, tt.Want.Fragment, u.Fragment)
		}

		if u.RawQuery != tt.Want.RawQuery {
			t.Errorf("%#v expected query %#v but go %#v", tt.Input, tt.Want.RawQuery, u.RawQuery)
		}
	}
}

type ProbeTest struct {
	Target         string
	Status         store.Status
	MessagePattern string
}

func AssertProbe(t *testing.T, tests []ProbeTest) {
	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			p := testutil.NewProbe(t, tt.Target)

			if p.Target().String() != tt.Target {
				t.Fatalf("got unexpected probe: %s", p.Target())
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			rs := testutil.RunCheck(ctx, p)

			if len(rs) != 1 {
				t.Fatalf("got unexpected number of results: %d", len(rs))
			}

			r := rs[0]
			if r.Target.String() != tt.Target {
				t.Errorf("got a record of unexpected target: %s", r.Target)
			}
			if r.Status != tt.Status {
				t.Errorf("expected status is %s but got %s", tt.Status, r.Status)
			}
			if ok, _ := regexp.MatchString("^"+tt.MessagePattern+"$", r.Message); !ok {
				t.Errorf("expected message is match to %#v but got %#v", tt.MessagePattern, r.Message)
			}
		})
	}
}

func AssertTimeout(t *testing.T, target string) {
	t.Run("timeout", func(t *testing.T) {
		p := testutil.NewProbe(t, target)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		defer cancel()

		records := testutil.RunCheck(ctx, p)
		if len(records) != 1 {
			t.Fatalf("unexpected number of records: %#v", records)
		}

		if records[0].Message != "probe timed out" {
			t.Errorf("unexpected message: %s", records[0].Message)
		}

		if records[0].Status != store.STATUS_UNKNOWN {
			t.Errorf("unexpected status: %s", records[0].Status)
		}
	})
	t.Run("cancel", func(t *testing.T) {
		p := testutil.NewProbe(t, target)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		records := testutil.RunCheck(ctx, p)
		if len(records) != 1 {
			t.Fatalf("unexpected number of records: %#v", records)
		}

		if records[0].Message != "probe aborted" {
			t.Errorf("unexpected message: %s", records[0].Message)
		}

		if records[0].Status != store.STATUS_ABORTED {
			t.Errorf("unexpected status: %s", records[0].Status)
		}
	})
}

func RunDummyHTTPServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})
	mux.HandleFunc("/redirect/ok", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ok", http.StatusFound)
	})
	mux.HandleFunc("/redirect/error", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/error", http.StatusFound)
	})
	mux.HandleFunc("/redirect/loop", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect/loop", http.StatusFound)
	})
	mux.HandleFunc("/only/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/only/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/only/head", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/only/options", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "OPTIONS" {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/slow-page", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("OK"))
	})

	return httptest.NewServer(mux)
}
