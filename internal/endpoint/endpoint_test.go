package endpoint_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/macrat/ayd/internal/testutil"
)

func TestStaticFiles(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/favicon.ico"); err != nil {
		t.Errorf("failed to get /favicon.ico: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}

	if resp, err := srv.Client().Get(srv.URL + "/favicon.svg"); err != nil {
		t.Errorf("failed to get /favicon.svg: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestNotFound(t *testing.T) {
	srv := testutil.StartTestServer(t)
	defer srv.Close()

	if resp, err := srv.Client().Get(srv.URL + "/not-found"); err != nil {
		t.Errorf("failed to get /not-found: %s", err)
	} else if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func readTestFile(t *testing.T, file string) string {
	t.Helper()

	f, err := os.Open(file)
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}

	return string(bs)
}

func AssertEndpoint(t *testing.T, endpoint, expectFile, maskPattern string) {
	t.Helper()

	srv := testutil.StartTestServer(t)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + endpoint)
	if err != nil {
		t.Fatalf("failed to get %s: %s", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %s", err)
	}

	result := strings.ReplaceAll(string(body), "\r\n", "\n")

	if maskPattern != "" {
		re, err := regexp.Compile(maskPattern)
		if err != nil {
			t.Fatalf("faield to parse mask pattern: %s", err)
		}

		result = re.ReplaceAllString(result, "[[MASKED_DATA]]")
	}

	if diff := cmp.Diff(readTestFile(t, expectFile), result); diff != "" {
		t.Errorf(diff)

		os.MkdirAll("./testdata/actual", 0755)
		f, err := os.Create(filepath.Join("./testdata/actual", filepath.Base(expectFile)))
		if err != nil {
			t.Fatalf("failed to create actual file: %s", err)
		}
		_, err = f.Write([]byte(result))
		if err != nil {
			t.Fatalf("failed to write actual file: %s", err)
		}
	}
}
