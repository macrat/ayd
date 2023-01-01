package endpoint

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type DummyFlushResponseWriter struct {
	Chunks  []string
	writing bool
}

func (f *DummyFlushResponseWriter) Header() http.Header {
	return http.Header{}
}

func (f *DummyFlushResponseWriter) Write(b []byte) (int, error) {
	if f.writing {
		idx := len(f.Chunks) - 1
		f.Chunks[idx] = f.Chunks[idx] + string(b)
	} else {
		f.Chunks = append(f.Chunks, string(b))
		f.writing = true
	}
	return len(b), nil
}

func (f *DummyFlushResponseWriter) WriteHeader(statusCode int) {
}

func (f *DummyFlushResponseWriter) Flush() {
	f.writing = false
}

func TestFlushWriter(t *testing.T) {
	responseChunkSize = 5

	dw := &DummyFlushResponseWriter{}
	fw := newFlushWriter(dw)

	assertW := func(want int) func(int, error) {
		return func(actual int, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if want != actual {
				t.Fatalf("expected length=%d but got length=%d", want, actual)
			}
		}
	}
	assertC := func(want ...string) {
		t.Helper()
		if diff := cmp.Diff(want, dw.Chunks); diff != "" {
			t.Fatalf("unexpected chunks found:\n%s", diff)
		}
	}

	assertC()

	assertW(5)(fw.Write([]byte("hello")))
	assertC("hello")

	assertW(5)(fw.Write([]byte("world")))
	assertC("hello", "world")

	assertW(6)(fw.Write([]byte("foobar")))
	assertC("hello", "world", "foobar")
}
