package endpoint

import (
	"io"
	"net/http"
)

var (
	responseChunkSize = 1024
)

type flushWriter struct {
	w http.ResponseWriter
	f http.Flusher

	count int
}

func newFlushWriter(w http.ResponseWriter) io.Writer {
	f, ok := w.(http.Flusher)
	if !ok {
		return w
	}
	return flushWriter{
		w: w,
		f: f,
	}
}

func (w flushWriter) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)

	w.count += n
	if w.count >= responseChunkSize {
		w.f.Flush()
		w.count = 0
	}

	return n, err
}
