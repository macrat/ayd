package probe_test

import (
	"net/url"
	"testing"

	"github.com/macrat/ayd/probe"
)

func TestTargetURLNormalize(t *testing.T) {
	tests := []struct {
		Input string
		Want  url.URL
	}{
		{"example.com", url.URL{Scheme: "ping", Opaque: "example.com"}},

		{"ping:example.com", url.URL{Scheme: "ping", Opaque: "example.com"}},
		{"ping://example.com:123/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "ping", Opaque: "example.com"}},

		{"http://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "http", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"https://example.com/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "https", Host: "example.com", Path: "/foo/bar", RawQuery: "hoge=fuga", Fragment: "piyo"}},

		{"tcp:example.com:80", url.URL{Scheme: "tcp", Opaque: "example.com:80"}},
		{"tcp://example.com:80/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "tcp", Opaque: "example.com:80"}},

		{"dns:example.com", url.URL{Scheme: "dns", Opaque: "example.com"}},
		{"dns://example.com:80/foo/bar?hoge=fuga#piyo", url.URL{Scheme: "dns", Opaque: "example.com"}},

		{"exec:foo.sh", url.URL{Scheme: "exec", Path: "foo.sh"}},
		{"exec:./foo.sh", url.URL{Scheme: "exec", Path: "./foo.sh"}},
		{"exec:/foo/bar.sh", url.URL{Scheme: "exec", Path: "/foo/bar.sh"}},
		{"exec:///foo/bar.sh", url.URL{Scheme: "exec", Path: "/foo/bar.sh"}},
		{"exec:foo.sh?hoge=fuga#piyo", url.URL{Scheme: "exec", Path: "foo.sh", RawQuery: "hoge=fuga", Fragment: "piyo"}},
		{"exec:/foo/bar.sh?hoge=fuga#piyo", url.URL{Scheme: "exec", Path: "/foo/bar.sh", RawQuery: "hoge=fuga", Fragment: "piyo"}},
	}

	for _, tt := range tests {
		p, err := probe.Get(tt.Input)
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
