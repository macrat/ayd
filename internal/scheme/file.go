package scheme

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
)

type proberFS interface {
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

type alerterFS interface {
	proberFS

	OpenAppend(name string, perm fs.FileMode) (io.WriteCloser, error)
}

type localFS struct{}

func (dir localFS) OpenAppend(name string, perm fs.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
}

func (dir localFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (dir localFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func probeFS[FS proberFS](dir FS, path string, r *api.Record) {
	stat, err := dir.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		r.Message = "no such file or directory"
		r.Status = api.StatusFailure
	case errors.Is(err, fs.ErrPermission):
		r.Message = "permission denied"
		r.Status = api.StatusFailure
	case err != nil:
		r.Message = fmt.Sprintf("failed to get information: %s", err)
		r.Status = api.StatusUnknown
	default:
		r.Status = api.StatusHealthy
		r.Extra = map[string]interface{}{
			"mtime":      stat.ModTime().Format(time.RFC3339),
			"permission": fmt.Sprintf("%03o", stat.Mode().Perm()),
		}
		if stat.IsDir() {
			r.Message = "directory exists"
			r.Extra["type"] = "directory"
			if entries, err := dir.ReadDir(path); err == nil {
				r.Extra["file_count"] = len(entries)
			}
		} else {
			r.Message = "file exists"
			r.Extra["type"] = "file"
			r.Extra["file_size"] = stat.Size()
		}
	}
	r.Latency = time.Since(r.Time)
}

func alertFS[FS alerterFS](dir FS, path string, r *api.Record, lastRecord api.Record) {
	f, err := dir.OpenAppend(path, 0600)
	if err != nil {
		r.Status = api.StatusFailure
		r.Message = fmt.Sprintf("failed to open target file: %s", err)
		return
	}
	defer f.Close()

	data, err := lastRecord.MarshalJSON()
	if err != nil {
		r.Status = api.StatusFailure
		r.Message = fmt.Sprintf("failed to convert record: %s", err)
		return
	}

	_, err = f.Write(append(data, '\n'))
	if err != nil {
		r.Status = api.StatusFailure
		r.Message = fmt.Sprintf("failed to write record: %s", err)
		return
	}

	r.Status = api.StatusHealthy
	r.Message = fmt.Sprintf("wrote %d bytes to file", len(data)+1)
}

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
	rec := api.Record{
		Time:   time.Now(),
		Target: s.target,
	}

	probeFS(localFS{}, s.target.Opaque, &rec)

	r.Report(s.target, timeoutOr(ctx, rec))
}

func (s FileScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	r = AlertReporter{s.target, r}

	rec := api.Record{
		Time:   time.Now(),
		Target: s.target,
	}

	alertFS(localFS{}, s.target.Opaque, &rec, lastRecord)

	r.Report(s.target, timeoutOr(ctx, rec))
}
