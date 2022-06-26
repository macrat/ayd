package scheme

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

// FileScheme is a probe/alert implementation for local file.
type FileScheme struct {
	target *api.URL
}

func NewFileScheme(u *api.URL) (FileScheme, error) {
	s := FileScheme{}

	path := u.Opaque
	if u.Opaque == "" {
		path = u.Path
	}
	if path == "" {
		path = "/"
	}
	s.target = &api.URL{
		Scheme:   "file",
		Opaque:   filepath.ToSlash(path),
		Fragment: u.Fragment,
	}

	return s, nil
}

func (s FileScheme) Target() *api.URL {
	return s.target
}

func (s FileScheme) Probe(ctx context.Context, r Reporter) {
	var status api.Status
	var message string
	var extra map[string]interface{}

	stime := time.Now()

	stat, err := os.Stat(s.target.Opaque)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		message = "no such file or directory"
		status = api.StatusFailure
	case errors.Is(err, fs.ErrPermission):
		message = "permission denied"
		status = api.StatusFailure
	case err != nil:
		message = fmt.Sprintf("failed to get information: %s", err)
		status = api.StatusUnknown
	default:
		status = api.StatusHealthy
		extra = map[string]interface{}{
			"mtime":      stat.ModTime().Format(time.RFC3339),
			"permission": fmt.Sprintf("%03o", stat.Mode().Perm()),
		}
		if stat.IsDir() {
			message = "directory exists"
			extra["type"] = "directory"
			if entries, err := os.ReadDir(s.target.Opaque); err == nil {
				extra["file_count"] = len(entries)
			}
		} else {
			message = "file exists"
			extra["type"] = "file"
			extra["file_size"] = stat.Size()
		}
	}

	r.Report(s.target, timeoutOr(ctx, api.Record{
		Time:    stime,
		Status:  status,
		Latency: time.Since(stime),
		Target:  s.target,
		Message: message,
		Extra:   extra,
	}))
}

func (s FileScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	r = AlertReporter{s.target, r}

	report := func(status api.Status, message string) {
		r.Report(s.target, timeoutOr(ctx, api.Record{
			Time:    time.Now(),
			Status:  status,
			Target:  s.target,
			Message: message,
		}))
	}

	f, err := os.OpenFile(s.target.Opaque, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		report(api.StatusFailure, fmt.Sprintf("failed to open target file: %s", err))
		return
	}
	defer f.Close()

	data, err := lastRecord.MarshalJSON()
	if err != nil {
		report(api.StatusFailure, fmt.Sprintf("failed to convert record: %s", err))
		return
	}

	_, err = f.Write(data)
	if err != nil {
		report(api.StatusFailure, fmt.Sprintf("failed to write record: %s", err))
		return
	}
	f.Write([]byte("\n"))

	report(api.StatusHealthy, fmt.Sprintf("wrote %d bytes to file", len(data)+1))
}
