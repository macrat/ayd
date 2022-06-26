package scheme_test

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"
	"testing"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	ftp "goftp.io/server/core"
)

type FTPFileInfo struct {
	name string
	size int64
	dir  bool
}

func (i FTPFileInfo) Name() string {
	return i.name
}

func (i FTPFileInfo) Size() int64 {
	return i.size
}

func (i FTPFileInfo) Mode() fs.FileMode {
	return 0644
}

func (i FTPFileInfo) ModTime() time.Time {
	// Year part is not work correctly because of the library's bug.
	// The server sets year part as current year, and drops seconds part.
	// But it still enough for the test.
	return time.Date(time.Now().Year(), 1, 2, 15, 4, 0, 0, time.UTC)
}

func (i FTPFileInfo) IsDir() bool {
	return i.dir
}

func (i FTPFileInfo) Sys() interface{} {
	return nil
}

func (i FTPFileInfo) Owner() string {
	return "hoge"
}

func (i FTPFileInfo) Group() string {
	return "hoge"
}

type FTPTestDriver struct{}

func (d FTPTestDriver) Stat(path string) (ftp.FileInfo, error) {
	switch path {
	case "/":
		return FTPFileInfo{"", 0, true}, nil
	case "/path":
		return FTPFileInfo{"path", 0, true}, nil
	case "/path/to":
		return FTPFileInfo{"to", 0, true}, nil
	case "/path/to/file.txt":
		return FTPFileInfo{"file.txt", 123, false}, nil
	case "/path/to/hello-world":
		return FTPFileInfo{"hello-world", 4321, false}, nil
	case "/slow-file":
		time.Sleep(2 * time.Second)
		return FTPFileInfo{"slow-file", 10, false}, nil
	}
	return nil, errors.New("no such file")
}

func (d FTPTestDriver) ListDir(path string, f func(ftp.FileInfo) error) error {
	switch path {
	case "/":
		f(FTPFileInfo{".", 0, true})
		f(FTPFileInfo{"to", 0, true})
	case "/path":
		f(FTPFileInfo{".", 0, true})
		f(FTPFileInfo{"..", 0, true})
		f(FTPFileInfo{"to", 0, true})
	case "/path/to":
		f(FTPFileInfo{".", 0, true})
		f(FTPFileInfo{"..", 0, true})
		f(FTPFileInfo{"file.txt", 123, false})
		f(FTPFileInfo{"hello-world", 4321, false})
	case "/path/to/file.txt":
		f(FTPFileInfo{"file.txt", 123, false})
	case "/path/to/hello-world":
		f(FTPFileInfo{"hello-world", 4321, false})
	case "/slow-file":
		f(FTPFileInfo{"flow-file", 10, false})
	}
	return nil
}

func (d FTPTestDriver) DeleteDir(path string) error {
	return nil
}

func (d FTPTestDriver) DeleteFile(path string) error {
	return nil
}

func (d FTPTestDriver) Rename(x, y string) error {
	return nil
}

func (d FTPTestDriver) MakeDir(path string) error {
	return nil
}

//go:embed testdata/healthy-list.txt
var healthySourceList []byte

func (d FTPTestDriver) GetFile(path string, i int64) (int64, io.ReadCloser, error) {
	if path == "/source.txt" {
		return int64(len(healthySourceList)), io.NopCloser(bytes.NewBuffer(healthySourceList)), nil
	}
	return 0, nil, errors.New("not implemented")
}

func (d FTPTestDriver) PutFile(path string, f io.Reader, b bool) (int64, error) {
	return 0, errors.New("not implemented")
}

func (d FTPTestDriver) NewDriver() (ftp.Driver, error) {
	return d, nil
}

type FTPTestAuth struct{}

func (a FTPTestAuth) CheckPasswd(username, password string) (ok bool, err error) {
	if username == "hoge" && password == "fuga" {
		return true, nil
	}
	if username == "anonymous" && password == "anonymous" {
		return true, nil
	}
	return false, nil
}

// StartFTPServer starts FTP server for test.
//
// XXX: randomize port and avoid conflict
func StartFTPServer(t *testing.T, port int) *ftp.Server {
	t.Helper()
	server := ftp.NewServer(&ftp.ServerOpts{
		Factory: FTPTestDriver{},
		Auth:    FTPTestAuth{},
		Port:    port,
		Logger:  &ftp.DiscardLogger{},
	})
	go func() {
		if err := server.ListenAndServe(); err != nil {
			t.Fatalf("failed to start ftp server: %s", err)
		}
		t.Cleanup(func() {
			server.Shutdown()
		})
	}()
	return server
}

func TestFTPProbe(t *testing.T) {
	t.Parallel()
	StartFTPServer(t, 21021)

	// See also the comment of FTPFileInfo.ModTime.
	mtime := fmt.Sprintf("%d-01-02T15:04:00Z", time.Now().Year())

	AssertProbe(t, []ProbeTest{
		{"ftp://localhost:21021/", api.StatusHealthy, "directory exists\n---\nfile_count: 1\nmtime: " + mtime + "\ntype: directory", ""},
		{"ftp://hoge:fuga@localhost:21021/", api.StatusHealthy, "directory exists\n---\nfile_count: 1\nmtime: " + mtime + "\ntype: directory", ""},
		{"ftp://foo:bar@localhost:21021/", api.StatusFailure, "530 Incorrect password, not logged in", ""},
		{"ftp://localhost:21021/path/to", api.StatusHealthy, "directory exists\n---\nfile_count: 2\nmtime: " + mtime + "\ntype: directory", ""},
		{"ftp://localhost:21021/path/to/file.txt", api.StatusHealthy, "file exists\n---\nfile_size: 123\nmtime: " + mtime + "\ntype: file", ""},
		{"ftp://localhost:21021/no/such/file.txt", api.StatusFailure, "550 no such file", ""},
		{"ftp://localhost:21021/slow-file", api.StatusFailure, "probe timed out", ""},

		{"ftps://localhost:21021/", api.StatusFailure, "550 Action not taken", ""},

		{"ftp:///without-host", api.StatusUnknown, ``, "missing target host"},
		{"ftp://hoge@localhost", api.StatusUnknown, ``, "password is required if set username"},
		{"ftp://:fuga@localhost", api.StatusUnknown, ``, "username is required if set password"},
	}, 1)

	if runtime.GOOS != "windows" {
		// Windows doesn't report connection refused. Why?
		AssertProbe(t, []ProbeTest{
			{"ftp://localhost:12345/", api.StatusFailure, `(127\.0\.0\.1|\[::1\]):12345: connection refused`, ""},
		}, 1)
	}
}
