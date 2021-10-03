package probe_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/probe"
	"github.com/macrat/ayd/internal/testutil"
	api "github.com/macrat/ayd/lib-ayd"
)

func TestTargetURLNormalize(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %s", err)
	}
	cwd = filepath.ToSlash(cwd)

	server := RunDummyHTTPServer()
	defer server.Close()

	tests := []struct {
		Input string
		Want  url.URL
		Error error
	}{
		{"ping:example.com", url.URL{Scheme: "ping", Opaque: "example.com"}, nil},
		{"ping://example.com:123/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "ping", Opaque: "example.com", Fragment: "piyo"}, nil},
		{"ping:example.com#piyo", url.URL{Scheme: "ping", Opaque: "example.com", Fragment: "piyo"}, nil},
		{"ping-abc:example.com", url.URL{Scheme: "ping", Opaque: "example.com"}, nil},

		{"http://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},

		{"http-get://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http-get", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https-post://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https-post", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"http-head://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http-head", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https-options://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https-options", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},

		{"tcp:example.com:80", url.URL{Scheme: "tcp", Host: "example.com:80"}, nil},
		{"tcp://example.com:80/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "tcp", Host: "example.com:80", Fragment: "piyo"}, nil},
		{"tcp4:example.com:80", url.URL{Scheme: "tcp4", Host: "example.com:80"}, nil},
		{"tcp6:example.com:80", url.URL{Scheme: "tcp6", Host: "example.com:80"}, nil},
		{"tcp:example.com:80#hello", url.URL{Scheme: "tcp", Host: "example.com:80", Fragment: "hello"}, nil},
		{"tcp-abc:example.com:80", url.URL{Scheme: "tcp", Host: "example.com:80"}, nil},

		{"dns:example.com", url.URL{Scheme: "dns", Opaque: "example.com"}, nil},
		{"dns:///example.com", url.URL{Scheme: "dns", Opaque: "example.com"}, nil},
		{"dns://8.8.8.8/example.com", url.URL{Scheme: "dns", Host: "8.8.8.8", Path: "/example.com"}, nil},
		{"dns://8.8.8.8:53/example.com", url.URL{Scheme: "dns", Host: "8.8.8.8:53", Path: "/example.com"}, nil},
		{"dns://example.com:53/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "dns", Host: "example.com:53", Path: "/foo", Fragment: "piyo"}, nil},
		{"dns:example.com#piyo", url.URL{Scheme: "dns", Opaque: "example.com", Fragment: "piyo"}, nil},

		{"dns:example.com?type=a&hoge=fuga", url.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=A"}, nil},
		{"dns-aaaa:example.com", url.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=AAAA"}, nil},
		{"dns-cname:example.com?type=TXT", url.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=CNAME"}, nil},

		{"exec:testdata/test.bat", url.URL{Scheme: "exec", Opaque: "testdata/test.bat"}, nil},
		{"exec:./testdata/test.bat", url.URL{Scheme: "exec", Opaque: "./testdata/test.bat"}, nil},
		{"exec:" + cwd + "/testdata/test.bat", url.URL{Scheme: "exec", Opaque: cwd + "/testdata/test.bat"}, nil},
		{"exec:testdata/test.bat?hoge=fuga#piyo", url.URL{Scheme: "exec", Opaque: "testdata/test.bat", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"exec-abc:testdata/test.bat", url.URL{Scheme: "exec", Opaque: "testdata/test.bat"}, nil},

		{"source:./testdata/healthy-list.txt", url.URL{Scheme: "source", Opaque: "./testdata/healthy-list.txt"}, nil},
		{"source:./testdata/healthy-list.txt#hello", url.URL{Scheme: "source", Opaque: "./testdata/healthy-list.txt", Fragment: "hello"}, nil},
		{"source-abc:./testdata/healthy-list.txt", url.URL{}, probe.ErrUnsupportedScheme},
		{"source+abc:./testdata/healthy-list.txt", url.URL{}, probe.ErrUnsupportedScheme},

		{"source-" + server.URL + "/source", url.URL{}, probe.ErrUnsupportedScheme},
		{"source+" + server.URL + "/source", url.URL{Scheme: "source+http", Host: strings.Replace(server.URL, "http://", "", 1), Path: "/source"}, nil},
		{"source+" + server.URL + "/error", url.URL{}, probe.ErrInvalidSource},
		{"source+https://of-course-no-such-host/source", url.URL{}, probe.ErrInvalidSource},

		{"source+exec:./testdata/listing-script", url.URL{Scheme: "source+exec", Opaque: "./testdata/listing-script"}, nil},
	}

	for _, tt := range tests {
		p, err := probe.New(tt.Input)
		if err != nil {
			if !errors.Is(err, tt.Error) {
				t.Errorf("%#v: unexpected error during create probe: %#s", tt.Input, err)
			}
			continue
		} else if tt.Error != nil {
			t.Errorf("%#v: expected error %#v but got nil", tt.Input, tt.Error)
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

	t.Run("unknown:target", func(t *testing.T) {
		_, err := probe.New("unknown:target")
		if err != probe.ErrUnsupportedScheme {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("abc", func(t *testing.T) {
		_, err := probe.New("abc")
		if err != probe.ErrMissingScheme {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("::", func(t *testing.T) {
		_, err := probe.New("::")
		if err != probe.ErrInvalidURL {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

type ProbeTest struct {
	Target            string
	Status            api.Status
	MessagePattern    string
	ParseErrorPattern string
}

func AssertProbe(t *testing.T, tests []ProbeTest) {
	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			p, err := probe.New(tt.Target)
			if err != nil {
				if ok, _ := regexp.MatchString("^"+tt.ParseErrorPattern+"$", err.Error()); !ok {
					t.Fatalf("unexpected error on create probe: %s", err)
				}
				return
			} else if tt.ParseErrorPattern != "" {
				t.Fatal("expected error on create probe but got nil")
			}

			if p.Target().String() != tt.Target {
				t.Fatalf("got unexpected probe: %s", p.Target())
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			rs := testutil.RunCheck(ctx, p)

			if len(rs) != 1 {
				t.Fatalf("got unexpected number of results: %d\n%v", len(rs), rs)
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

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		records := testutil.RunCheck(ctx, p)
		if len(records) != 1 {
			t.Fatalf("unexpected number of records: %#v", records)
		}

		if records[0].Message != "probe timed out" {
			t.Errorf("unexpected message: %s", records[0].Message)
		}

		if records[0].Status != api.StatusFailure {
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

		if records[0].Status != api.StatusAborted {
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
	mux.HandleFunc("/source", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dummy:healthy#1\ndummy:healthy#2"))
	})
	mux.HandleFunc("/source/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dummy:healthy#1\ndummy:healthy#2"))
	})

	return httptest.NewServer(mux)
}
