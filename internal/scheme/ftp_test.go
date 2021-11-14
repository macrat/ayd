package scheme_test

import (
	"errors"
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
	return time.Now()
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

func (d FTPTestDriver) GetFile(path string, i int64) (int64, io.ReadCloser, error) {
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

func StartFTPServer(t *testing.T) *ftp.Server {
	t.Helper()
	server := ftp.NewServer(&ftp.ServerOpts{
		Factory: FTPTestDriver{},
		Auth:    FTPTestAuth{},
		Port:    21021,
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
	StartFTPServer(t)

	AssertProbe(t, []ProbeTest{
		{"ftp://localhost:21021/", api.StatusHealthy, `type=directory files=1`, ""},
		{"ftp://hoge:fuga@localhost:21021/", api.StatusHealthy, `type=directory files=1`, ""},
		{"ftp://foo:bar@localhost:21021/", api.StatusFailure, `530 Incorrect password, not logged in`, ""},
		{"ftp://localhost:21021/path/to", api.StatusHealthy, `type=directory files=2`, ""},
		{"ftp://localhost:21021/path/to/file.txt", api.StatusHealthy, `type=file size=123`, ""},
		{"ftp://localhost:21021/no/such/file.txt", api.StatusFailure, `550 no such file`, ""},
		{"ftp://localhost:21021/slow-file", api.StatusFailure, `probe timed out`, ""},

		{"ftps://localhost:21021/", api.StatusFailure, `550 Action not taken`, ""},

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
