package scheme

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/pkg/sftp"
)

type sftpFS struct {
	ssh  sshConnection
	sftp *sftp.Client
}

func (dir sftpFS) Close() error {
	dir.sftp.Close()
	return dir.ssh.Close()
}

func (dir sftpFS) OpenAppend(name string, perm fs.FileMode) (io.WriteCloser, error) {
	return dir.sftp.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
}

func (dir sftpFS) Stat(name string) (fs.FileInfo, error) {
	return dir.sftp.Stat(name)
}

func (dir sftpFS) ReadDir(name string) ([]fs.DirEntry, error) {
	fis, err := dir.sftp.ReadDir(name)
	if err != nil {
		return nil, err
	}
	var ds []fs.DirEntry
	for _, f := range fis {
		ds = append(ds, fs.FileInfoToDirEntry(f))
	}
	return ds, err
}

type SFTPScheme struct {
	target *api.URL
}

func NewSFTPScheme(u *api.URL) (SFTPScheme, error) {
	_, separator, _ := SplitScheme(u.Scheme)
	if separator != 0 {
		return SFTPScheme{}, ErrUnsupportedScheme
	}

	q := url.Values{}
	if f := u.ToURL().Query().Get("identityfile"); f != "" {
		q.Set("identityfile", f)
	}
	if f := u.ToURL().Query().Get("fingerprint"); f != "" {
		q.Set("fingerprint", strings.ReplaceAll(f, " ", "+"))
	}
	u.RawQuery = q.Encode()

	_, err := newSSHConfig(u)
	if err != nil {
		return SFTPScheme{}, err
	}

	env := make(map[string]string)
	for k, v := range u.ToURL().Query() {
		env[k] = v[len(v)-1]
	}

	return SFTPScheme{
		target: u,
	}, nil
}

func (s SFTPScheme) Target() *api.URL {
	return s.target
}

func (s SFTPScheme) dial(ctx context.Context, rec *api.Record) (sfs sftpFS, ok bool) {
	conf, err := newSSHConfig(s.target)
	if err != nil {
		rec.Status = api.StatusUnknown
		rec.Message = err.Error()
		return sfs, false
	}

	sfs.ssh, err = dialSSH(ctx, conf)
	rec.Extra = sfs.ssh.MakeExtra()

	var sshErr sshError
	if errors.As(err, &sshErr) {
		rec.Status = sshErr.Status
		rec.Message = sshErr.Message
		return sfs, false
	} else if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = fmt.Sprintf("failed to connect: %s", err)
		return sfs, false
	}

	sfs.sftp, err = sftp.NewClient(sfs.ssh.Client)
	if err != nil {
		rec.Status = api.StatusFailure
		rec.Message = fmt.Sprintf("failed to establish SFTP connection: %s", err)
		return sfs, false
	}

	return sfs, true
}

func (s SFTPScheme) Probe(ctx context.Context, r Reporter) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	rec := api.Record{
		Time:   time.Now(),
		Target: s.target,
	}

	sfs, ok := s.dial(ctx, &rec)
	if !ok {
		rec.Latency = time.Since(rec.Time)
		r.Report(s.target, timeoutOr(ctx, rec))
		return
	}
	defer sfs.Close()
	rec.Extra = nil

	probeFS(sfs, s.target.Path, &rec)

	rec.Latency = time.Since(rec.Time)
	r.Report(s.target, timeoutOr(ctx, rec))
}

func (s SFTPScheme) Alert(ctx context.Context, r Reporter, lastRecord api.Record) {
	r = AlertReporter{s.target, r}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	rec := api.Record{
		Time:   time.Now(),
		Target: s.target,
	}

	sfs, ok := s.dial(ctx, &rec)
	if !ok {
		rec.Latency = time.Since(rec.Time)
		r.Report(s.target, timeoutOr(ctx, rec))
		return
	}
	defer sfs.Close()
	rec.Extra = nil

	alertFS(sfs, s.target.Path, &rec, lastRecord)

	rec.Latency = time.Since(rec.Time)
	r.Report(s.target, timeoutOr(ctx, rec))
}
