package scheme_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/macrat/ayd/internal/scheme"
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

	StartFTPServer(t, 2121)

	tests := []struct {
		Input string
		Want  api.URL
		Error error
	}{
		{"ping:example.com", api.URL{Scheme: "ping", Opaque: "example.com"}, nil},
		{"ping://example.com:123/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "ping", Opaque: "example.com", Fragment: "piyo"}, nil},
		{"ping:example.com#piyo", api.URL{Scheme: "ping", Opaque: "example.com", Fragment: "piyo"}, nil},
		{"PiNg:ExAmPlE.cOm", api.URL{Scheme: "ping", Opaque: "example.com"}, nil},
		{"ping-abc:example.com", api.URL{}, scheme.ErrUnsupportedScheme},
		{"ping+abc:example.com", api.URL{}, scheme.ErrUnsupportedScheme},
		{"ping:", api.URL{}, scheme.ErrMissingHost},
		{"ping:///test", api.URL{}, scheme.ErrMissingHost},

		{"http://example.com", api.URL{Scheme: "http", Host: "example.com", Path: "/"}, nil},
		{"http://example.com/", api.URL{Scheme: "http", Host: "example.com", Path: "/"}, nil},
		{"http://example.com/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "http", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https://example.com/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "https", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"HtTpS://eXaMpLe.CoM/fOo/BaR", api.URL{Scheme: "https", Host: "example.com", Path: "/fOo/BaR"}, nil},

		{"http-get://example.com/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "http-get", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https-post://example.com/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "https-post", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"http-head://example.com/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "http-head", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https-options://example.com/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "https-options", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"https+get://example.com", api.URL{}, scheme.ErrUnsupportedScheme},
		{"https:///test", api.URL{}, scheme.ErrMissingHost},
		{"https:", api.URL{}, scheme.ErrMissingHost},

		{"ftp://example.com", api.URL{Scheme: "ftp", Host: "example.com", Path: "/"}, nil},
		{"ftp://example.com/?abc=def", api.URL{Scheme: "ftp", Host: "example.com", Path: "/"}, nil},
		{"ftps://example.com/foo/bar/.././bar/", api.URL{Scheme: "ftps", Host: "example.com", Path: "/foo/bar"}, nil},

		{"tcp:example.com:80", api.URL{Scheme: "tcp", Host: "example.com:80"}, nil},
		{"tcp://example.com:80/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "tcp", Host: "example.com:80", Fragment: "piyo"}, nil},
		{"tcp4:example.com:80", api.URL{Scheme: "tcp4", Host: "example.com:80"}, nil},
		{"tcp6:example.com:80", api.URL{Scheme: "tcp6", Host: "example.com:80"}, nil},
		{"tcp:example.com:80#hello", api.URL{Scheme: "tcp", Host: "example.com:80", Fragment: "hello"}, nil},
		{"TcP:eXaMpLe.CoM:80", api.URL{Scheme: "tcp", Host: "example.com:80"}, nil},
		{"tcp-abc:example.com:80", api.URL{}, scheme.ErrUnsupportedScheme},
		{"tcp-def:example.com:80", api.URL{}, scheme.ErrUnsupportedScheme},
		{"tcp://:80", api.URL{}, scheme.ErrMissingHost},
		{"tcp://", api.URL{}, scheme.ErrMissingHost},
		{"tcp:", api.URL{}, scheme.ErrMissingHost},

		{"dns:example.com", api.URL{Scheme: "dns", Opaque: "example.com"}, nil},
		{"dns:///example.com", api.URL{Scheme: "dns", Opaque: "example.com"}, nil},
		{"dns://8.8.8.8/example.com", api.URL{Scheme: "dns", Host: "8.8.8.8", Path: "/example.com"}, nil},
		{"dns://8.8.8.8:53/example.com", api.URL{Scheme: "dns", Host: "8.8.8.8:53", Path: "/example.com"}, nil},
		{"dns://example.com:53/foo/bar?hoge=fuga#piyo", api.URL{Scheme: "dns", Host: "example.com:53", Path: "/foo", Fragment: "piyo"}, nil},
		{"dns:example.com#piyo", api.URL{Scheme: "dns", Opaque: "example.com", Fragment: "piyo"}, nil},
		{"DnS:lOcAlHoSt?TyPe=AaAa", api.URL{Scheme: "dns", Opaque: "localhost", RawQuery: "type=AAAA"}, nil},
		{"dns:", api.URL{}, scheme.ErrMissingDomainName},
		{"dns://8.8.8.8:53", api.URL{}, scheme.ErrMissingDomainName},

		{"dns:example.com?type=a&hoge=fuga", api.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=A"}, nil},
		{"dns-aaaa:example.com", api.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=AAAA"}, nil},
		{"dns4:example.com", api.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=A"}, nil},
		{"dns6:example.com", api.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=AAAA"}, nil},
		{"dns+a:example.com", api.URL{}, scheme.ErrUnsupportedScheme},
		{"dns-abc:example.com", api.URL{}, scheme.ErrUnsupportedDNSType},
		{"dns-cname:example.com?type=TXT", api.URL{}, scheme.ErrConflictDNSType},
		{"dns4:example.com?type=AAAA", api.URL{}, scheme.ErrConflictDNSType},
		{"dns-txt:example.com?type=TXT", api.URL{Scheme: "dns", Opaque: "example.com", RawQuery: "type=TXT"}, nil},

		{"exec:testdata/test.bat", api.URL{Scheme: "exec", Opaque: "testdata/test.bat"}, nil},
		{"exec:./testdata/test.bat", api.URL{Scheme: "exec", Opaque: "./testdata/test.bat"}, nil},
		{"exec:" + cwd + "/testdata/test.bat", api.URL{Scheme: "exec", Opaque: cwd + "/testdata/test.bat"}, nil},
		{"exec:testdata/test.bat?hoge=fuga#piyo", api.URL{Scheme: "exec", Opaque: "testdata/test.bat", RawQuery: "hoge=fuga", Fragment: "piyo"}, nil},
		{"exec-abc:testdata/test", api.URL{}, scheme.ErrUnsupportedScheme},
		{"exec+abc:testdata/test", api.URL{}, scheme.ErrUnsupportedScheme},
		{"exec:", api.URL{}, scheme.ErrMissingCommand},
		{"exec://", api.URL{}, scheme.ErrMissingCommand},

		{"source:./testdata/healthy-list.txt", api.URL{Scheme: "source", Opaque: "testdata/healthy-list.txt"}, nil},
		{"source:testdata/healthy-list.txt#hello", api.URL{Scheme: "source", Opaque: "testdata/healthy-list.txt", Fragment: "hello"}, nil},
		{"source-abc:./testdata/healthy-list.txt", api.URL{}, scheme.ErrUnsupportedScheme},
		{"source+abc:./testdata/healthy-list.txt", api.URL{}, scheme.ErrUnsupportedScheme},
		{"source:", api.URL{}, scheme.ErrMissingFile},
		{"source+http:", api.URL{}, scheme.ErrMissingHost},
		{"source+ftp:", api.URL{}, scheme.ErrMissingHost},
		{"source+ftps:", api.URL{}, scheme.ErrMissingHost},
		{"source+exec:", api.URL{}, scheme.ErrMissingCommand},
		{"source+ftp://example.com/", api.URL{}, scheme.ErrMissingFile},
		{"source+ftps://example.com/", api.URL{}, scheme.ErrMissingFile},

		{"source-" + server.URL + "/source", api.URL{}, scheme.ErrUnsupportedScheme},
		{"source+" + server.URL + "/source", api.URL{Scheme: "source+http", Host: strings.Replace(server.URL, "http://", "", 1), Path: "/source"}, nil},
		{"source+" + strings.ToUpper(server.URL) + "/source", api.URL{Scheme: "source+http", Host: strings.Replace(server.URL, "http://", "", 1), Path: "/source"}, nil},
		{"source+" + server.URL + "/error", api.URL{}, scheme.ErrInvalidSource},
		{"source+https://of-course-no-such-host/source", api.URL{}, scheme.ErrInvalidSource},
		{"source+" + server.URL + "/", api.URL{Scheme: "source+http", Host: strings.Replace(server.URL, "http://", "", 1), Path: "/"}, nil},
		{"source+" + server.URL, api.URL{Scheme: "source+http", Host: strings.Replace(server.URL, "http://", "", 1), Path: "/"}, nil},

		{"source+ftp://localhost:2121/source.txt", api.URL{Scheme: "source+ftp", Host: "localhost:2121", Path: "/source.txt"}, nil},

		{"source+exec:./testdata/listing-script", api.URL{Scheme: "source+exec", Opaque: "./testdata/listing-script"}, nil},
	}

	for _, tt := range tests {
		p, err := scheme.NewProber(tt.Input)
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
		_, err := scheme.NewProber("unknown:target")
		if err != scheme.ErrUnsupportedScheme {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("abc", func(t *testing.T) {
		_, err := scheme.NewProber("abc")
		if err != scheme.ErrMissingScheme {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("::", func(t *testing.T) {
		_, err := scheme.NewProber("::")
		if err != scheme.ErrInvalidURL {
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

func AssertProbe(t *testing.T, tests []ProbeTest, timeout int) {
	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			p, err := scheme.NewProber(tt.Target)
			if err != nil {
				if ok, _ := regexp.MatchString("^"+tt.ParseErrorPattern+"$", err.Error()); !ok {
					t.Fatalf("unexpected error on create probe: %s", err)
				}
				return
			} else if tt.ParseErrorPattern != "" {
				t.Fatal("expected error on create probe but got nil")
			}

			target := regexp.MustCompile(":[^:]*@").ReplaceAllString(tt.Target, ":xxxxx@")

			if p.Target().String() != target {
				t.Fatalf("got unexpected probe: expected %s but got %s", target, p.Target())
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()

			rs := testutil.RunProbe(ctx, p)

			if len(rs) != 1 {
				t.Fatalf("got unexpected number of results: %d\n%v", len(rs), rs)
			}

			r := rs[0]
			if r.Target.String() != target {
				t.Errorf("got a record of unexpected target: %s", r.Target)
			}
			if r.Status != tt.Status {
				t.Errorf("expected status is %s but got %s", tt.Status, r.Status)
			}
			if ok, _ := regexp.MatchString("^"+tt.MessagePattern+"$", r.ReadableMessage()); !ok {
				t.Errorf("unexpected message\n----- expected pattern -----\n%s\n----- actual -----\n%s", tt.MessagePattern, r.ReadableMessage())
			}
		})
	}
}

func AssertTimeout(t *testing.T, target string) {
	t.Run("timeout", func(t *testing.T) {
		p := testutil.NewProber(t, target)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		records := testutil.RunProbe(ctx, p)
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
		p := testutil.NewProber(t, target)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		records := testutil.RunProbe(ctx, p)
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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
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
	mux.HandleFunc("/only/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "CONNECT" {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/slow-page", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/source", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dummy:healthy#1\ndummy:healthy#2"))
	})
	mux.HandleFunc("/source/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dummy:healthy#1\ndummy:healthy#2"))
	})

	return httptest.NewServer(mux)
}
