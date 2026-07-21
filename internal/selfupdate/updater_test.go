package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		valid bool
	}{
		{value: "v0.1.0", valid: true},
		{value: "v12.34.56", valid: true},
		{value: "1.2.3"},
		{value: "v1.2"},
		{value: "v1.02.3"},
		{value: "v1.2.3-beta"},
	}
	for _, tt := range tests {
		_, err := parseVersion(tt.value)
		if (err == nil) != tt.valid {
			t.Errorf("parseVersion(%q) error = %v, valid = %t", tt.value, err, tt.valid)
		}
	}

	older, _ := parseVersion("v1.9.9")
	newer, _ := parseVersion("v2.0.0")
	if compareVersions(newer, older) <= 0 {
		t.Error("v2.0.0 should be newer than v1.9.9")
	}
	if compareVersions(older, older) != 0 {
		t.Error("equal versions should compare equally")
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/releases/latest" {
			http.NotFound(writer, request)
			return
		}
		_, _ = io.WriteString(writer, `{"tag_name":"v1.3.0"}`)
	}))
	defer server.Close()

	updater := New("v1.2.3")
	updater.APIBase = server.URL
	result, err := updater.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Current != "v1.2.3" || result.Latest != "v1.3.0" || !result.Updated {
		t.Errorf("Check() = %+v", result)
	}
}

func TestUpdateDownloadsVerifiesAndStagesRelease(t *testing.T) {
	t.Parallel()

	binary := []byte("new jup binary")
	archive := tarGzipArchive(t, map[string][]byte{
		"javaup-1.1.0-linux-amd64/jup":     binary,
		"javaup-1.1.0-linux-amd64/LICENSE": []byte("license"),
	})
	archiveName := "javaup-1.1.0-linux-amd64.tar.gz"
	digest := sha256.Sum256(archive)
	checksums := fmt.Sprintf("%x  %s\n", digest, archiveName)

	server := releaseServer(t, archiveName, archive, checksums)
	defer server.Close()
	target := filepath.Join(t.TempDir(), "jup")
	if err := os.WriteFile(target, []byte("old binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	applied := false
	updater := New("v1.0.0")
	updater.APIBase = server.URL
	updater.ReleaseBase = server.URL
	updater.GOOS = "linux"
	updater.GOARCH = "amd64"
	updater.ExecutablePath = target
	updater.apply = func(staged, gotTarget string) (bool, error) {
		applied = true
		gotTargetInfo, err := os.Stat(gotTarget)
		if err != nil {
			return false, err
		}
		targetInfo, err := os.Stat(target)
		if err != nil {
			return false, err
		}
		if !os.SameFile(gotTargetInfo, targetInfo) {
			t.Errorf("target %q and expected target %q are different files", gotTarget, target)
		}
		// #nosec G304 -- staged is created by the updater inside t.TempDir().
		got, err := os.ReadFile(staged)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(got, binary) {
			t.Errorf("staged binary = %q, want %q", got, binary)
		}
		return false, nil
	}

	result, err := updater.Update(context.Background())
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !applied || !result.Updated || result.Pending || result.Latest != "v1.1.0" {
		t.Errorf("Update() = %+v, applied = %t", result, applied)
	}
}

func TestUpdateRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	archiveName := "javaup-1.1.0-linux-amd64.tar.gz"
	archive := tarGzipArchive(t, map[string][]byte{"package/jup": []byte("binary")})
	server := releaseServer(t, archiveName, archive, strings.Repeat("0", 64)+"  "+archiveName+"\n")
	defer server.Close()
	target := filepath.Join(t.TempDir(), "jup")
	if err := os.WriteFile(target, []byte("old binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	updater := New("v1.0.0")
	updater.APIBase = server.URL
	updater.ReleaseBase = server.URL
	updater.GOOS = "linux"
	updater.GOARCH = "amd64"
	updater.ExecutablePath = target
	updater.apply = func(_, _ string) (bool, error) {
		t.Fatal("apply should not be called after a checksum mismatch")
		return false, nil
	}
	_, err := updater.Update(context.Background())
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Update() error = %v, want checksum mismatch", err)
	}
	// #nosec G304 -- target is a test-controlled path inside t.TempDir().
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "old binary" {
		t.Errorf("target was modified: %q", got)
	}
}

func releaseServer(t *testing.T, archiveName string, archive []byte, checksums string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/releases/latest":
			_, _ = io.WriteString(writer, `{"tag_name":"v1.1.0"}`)
		case "/download/v1.1.0/checksums.txt":
			_, _ = io.WriteString(writer, checksums)
		case "/download/v1.1.0/" + archiveName:
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
}

func tarGzipArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, contents := range files {
		if err := tarWriter.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(contents)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
