package textdecode

import (
	"io"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Reader makes new io.Reader to read data as unicode string.
func Reader(r io.Reader) io.Reader {
	return transform.NewReader(r, transform.Chain(
		unicode.BOMOverride(localeDecoder()),
		newlineNormalizer{},
	))
}

// ReadCloser is almost the same as Reader but it makes io.ReadCloser instead of io.Reader.
func ReadCloser(r io.ReadCloser) io.ReadCloser {
	return readCloser{
		Reader: Reader(r),
		Closer: r,
	}
}

type readCloser struct {
	Reader io.Reader
	Closer io.Closer
}

func (r readCloser) Read(b []byte) (int, error) {
	return r.Reader.Read(b)
}

func (r readCloser) Close() error {
	return r.Closer.Close()
}

// ToString decodes io.Reader to string.
func ToString(r io.Reader) (string, error) {
	x, err := io.ReadAll(Reader(r))
	if err != nil {
		return "", err
	}
	return string(x), nil
}
