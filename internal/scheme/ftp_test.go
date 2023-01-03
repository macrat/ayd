package scheme_test

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/scheme"
	"github.com/macrat/ayd/internal/testutil"
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

type FTPUploadedFile struct {
	Name   string
	Data   []byte
	Append bool
}

type FTPTestDriver struct {
	Uploaded []FTPUploadedFile
}

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

func (d *FTPTestDriver) PutFile(path string, f io.Reader, append_ bool) (int64, error) {
	if b, err := io.ReadAll(f); err != nil {
		return 0, err
	} else {
		d.Uploaded = append(d.Uploaded, FTPUploadedFile{
			Name:   path,
			Data:   b,
			Append: append_,
		})
		return int64(len(b)), nil
	}
}

func (d *FTPTestDriver) NewDriver() (ftp.Driver, error) {
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
func StartFTPServer(t *testing.T) (driver *FTPTestDriver, addr string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start ftp server: %s", err)
	}

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		listener.Close()
		t.Fatalf("failed to get port: %s", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		listener.Close()
		t.Fatalf("failed to parse port: %s", err)
	}

	driver = &FTPTestDriver{}

	server := ftp.NewServer(&ftp.ServerOpts{
		Factory: driver,
		Auth:    FTPTestAuth{},
		Port:    port,
		Logger:  &ftp.DiscardLogger{},
	})
	go func(listener net.Listener) {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Errorf("failed to start ftp server: %w", err))
		}
		t.Cleanup(func() {
			server.Shutdown()
			listener.Close()
		})
	}(listener)

	return driver, listener.Addr().String()
}

func TestFTPScheme_Probe(t *testing.T) {
	t.Parallel()
	_, addr := StartFTPServer(t)

	// See also the comment of FTPFileInfo.ModTime.
	mtime := fmt.Sprintf("%d-01-02T15:04:00Z", time.Now().Year())

	AssertProbe(t, []ProbeTest{
		{"ftp://" + addr + "/", api.StatusHealthy, "directory exists\n---\nfile_count: 1\nmtime: " + mtime + "\ntype: directory", ""},
		{"ftp://hoge:fuga@" + addr + "/", api.StatusHealthy, "directory exists\n---\nfile_count: 1\nmtime: " + mtime + "\ntype: directory", ""},
		{"ftp://foo:bar@" + addr + "/", api.StatusFailure, "530 Incorrect password, not logged in", ""},
		{"ftp://" + addr + "/path/to", api.StatusHealthy, "directory exists\n---\nfile_count: 2\nmtime: " + mtime + "\ntype: directory", ""},
		{"ftp://" + addr + "/path/to/file.txt", api.StatusHealthy, "file exists\n---\nfile_size: 123\nmtime: " + mtime + "\ntype: file", ""},
		{"ftp://" + addr + "/no/such/file.txt", api.StatusFailure, "550 no such file", ""},
		{"ftp://" + addr + "/slow-file", api.StatusFailure, "probe timed out", ""},

		{"ftps://" + addr + "/", api.StatusFailure, "550 Action not taken", ""},

		{"ftp:///without-host", api.StatusUnknown, ``, "missing target host"},
		{"ftp://hoge@localhost", api.StatusUnknown, ``, "password is required if set username"},
		{"ftp://:fuga@localhost", api.StatusUnknown, ``, "username is required if set password"},
	}, 1)

	if runtime.GOOS != "windows" {
		// Windows doesn't report connection refused. Why?
		AssertProbe(t, []ProbeTest{
			{"ftp://localhost:10/", api.StatusFailure, `(127\.0\.0\.1|\[::1\]):10: connection refused`, ""},
		}, 1)
	}
}

func TestFTPScheme_Alert(t *testing.T) {
	t.Parallel()

	driver, addr := StartFTPServer(t)

	a, err := scheme.NewAlerter("ftp://hoge:fuga@" + addr + "/alert.json")
	if err != nil {
		t.Fatalf("failed to prepare FTPScheme: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r := &testutil.DummyReporter{}
	a.Alert(ctx, r, api.Record{
		Time:    time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC),
		Status:  api.StatusFailure,
		Latency: 123456 * time.Microsecond,
		Target:  &api.URL{Scheme: "dummy", Fragment: "hello"},
		Message: "hello world",
	})

	expected := `{"time":"2021-01-02T15:04:05Z", "status":"FAILURE", "latency":123.456, "target":"dummy:#hello", "message":"hello world"}` + "\n"

	if len(r.Records) != 1 {
		t.Errorf("unexpected number of records\n%v", r.Records)
	} else {
		if r.Records[0].Status != api.StatusHealthy {
			t.Errorf("unexpected status: %s", r.Records[0].Status)
		}
		if r.Records[0].Message != fmt.Sprintf("uploaded %d bytes to the server", len(expected)) {
			t.Errorf("unexpected message: %q", r.Records[0].Message)
		}
	}

	if len(driver.Uploaded) != 1 {
		t.Errorf("unexpected number of uploaded files found: %d", len(driver.Uploaded))
	} else {
		info := driver.Uploaded[0]
		if info.Name != "/alert.json" {
			t.Errorf("unexpected name file uploaded: %s", info.Name)
		}
		if diff := cmp.Diff(expected, string(info.Data)); diff != "" {
			t.Errorf("unexpected file uploaded:\n%s", diff)
		}
		if !info.Append {
			t.Errorf("the append flag was false")
		}
	}
}
