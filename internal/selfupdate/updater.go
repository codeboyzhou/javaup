// Package selfupdate downloads and installs javaup releases.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/codeboyzhou/javaup/internal/pathutil"
)

const (
	defaultAPIBase     = "https://api.github.com/repos/codeboyzhou/javaup"
	defaultReleaseBase = "https://github.com/codeboyzhou/javaup/releases"
	defaultHTTPTimeout = 10 * time.Minute
	maxMetadataSize    = 1 << 20
	maxArchiveSize     = 128 << 20
	maxBinarySize      = 64 << 20
)

// Result describes the outcome of an update check or installation.
type Result struct {
	Current string
	Latest  string
	Updated bool
	Pending bool
}

// Updater checks GitHub Releases and replaces the current executable.
type Updater struct {
	CurrentVersion string
	HTTPClient     *http.Client
	APIBase        string
	ReleaseBase    string
	GOOS           string
	GOARCH         string
	ExecutablePath string
	apply          func(string, string) (bool, error)
}

// New returns an updater configured for the official javaup repository.
func New(currentVersion string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		HTTPClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		APIBase:     defaultAPIBase,
		ReleaseBase: defaultReleaseBase,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
		apply:       applyUpdate,
	}
}

// Check reports whether a newer release is available without downloading it.
func (u *Updater) Check(ctx context.Context) (Result, error) {
	current, err := parseVersion(u.CurrentVersion)
	if err != nil {
		return Result{}, fmt.Errorf("read current version: %w", err)
	}

	latestTag, err := u.latestVersion(ctx)
	if err != nil {
		return Result{}, err
	}
	latest, err := parseVersion(latestTag)
	if err != nil {
		return Result{}, fmt.Errorf("latest release: %w", err)
	}

	return Result{
		Current: current.tag,
		Latest:  latest.tag,
		Updated: compareVersions(latest, current) > 0,
	}, nil
}

// Update downloads, verifies, and installs the latest release when it is newer.
func (u *Updater) Update(ctx context.Context) (Result, error) {
	return u.UpdateWithProgress(ctx, nil)
}

// UpdateWithProgress downloads, verifies, and installs the latest release while reporting progress.
func (u *Updater) UpdateWithProgress(ctx context.Context, progress ProgressFunc) (Result, error) {
	reportProgress(progress, ProgressEvent{Stage: ProgressChecking})
	result, err := u.Check(ctx)
	if err != nil || !result.Updated {
		return result, err
	}
	reportProgress(progress, ProgressEvent{
		Stage: ProgressAvailable, Current: result.Current, Latest: result.Latest,
	})
	if err := validatePlatform(u.GOOS, u.GOARCH); err != nil {
		return Result{}, err
	}

	target := u.ExecutablePath
	if target == "" {
		target, err = os.Executable()
		if err != nil {
			return Result{}, fmt.Errorf("locate current executable: %w", err)
		}
	}
	target, err = pathutil.Resolved(target)
	if err != nil {
		return Result{}, fmt.Errorf("resolve current executable: %w", err)
	}

	archiveName := releaseArchiveName(result.Latest, u.GOOS, u.GOARCH)
	downloadBase := strings.TrimRight(u.ReleaseBase, "/") + "/download/" + result.Latest
	checksumsURL := downloadBase + "/checksums.txt"
	reportProgress(progress, ProgressEvent{Stage: ProgressFetchingChecksums, URL: checksumsURL})
	checksums, err := u.download(ctx, checksumsURL, maxMetadataSize)
	if err != nil {
		return Result{}, fmt.Errorf("download checksums: %w", err)
	}
	expected, err := expectedChecksum(checksums, archiveName)
	if err != nil {
		return Result{}, err
	}
	archiveURL := downloadBase + "/" + archiveName
	reportProgress(progress, ProgressEvent{Stage: ProgressConnecting, Name: archiveName, URL: archiveURL})
	archive, err := u.downloadWithProgress(
		ctx,
		archiveURL,
		maxArchiveSize,
		func(downloaded, total int64) {
			reportProgress(progress, ProgressEvent{
				Stage: ProgressDownloading, Name: archiveName, Downloaded: downloaded, Total: total,
			})
		},
	)
	if err != nil {
		return Result{}, fmt.Errorf("download %s: %w", archiveName, err)
	}
	actual := sha256.Sum256(archive)
	if !strings.EqualFold(expected, hex.EncodeToString(actual[:])) {
		return Result{}, fmt.Errorf("checksum mismatch for %s", archiveName)
	}
	reportProgress(progress, ProgressEvent{Stage: ProgressVerifying, Name: archiveName})

	reportProgress(progress, ProgressEvent{Stage: ProgressExtracting, Name: archiveName})
	staged, err := stageBinary(archive, u.GOOS, target)
	if err != nil {
		return Result{}, err
	}
	keepStaged := false
	defer func() {
		if !keepStaged {
			_ = os.Remove(staged)
		}
	}()

	reportProgress(progress, ProgressEvent{Stage: ProgressInstalling, Name: filepath.Base(target)})
	pending, err := u.apply(staged, target)
	if err != nil {
		return Result{}, fmt.Errorf("replace current executable: %w", err)
	}
	keepStaged = pending
	result.Pending = pending
	return result, nil
}

func (u *Updater) latestVersion(ctx context.Context) (string, error) {
	data, err := u.download(ctx, strings.TrimRight(u.APIBase, "/")+"/releases/latest", maxMetadataSize)
	if err != nil {
		return "", fmt.Errorf("query latest release: %w", err)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if release.TagName == "" {
		return "", errors.New("latest release response does not contain a tag")
	}
	return release.TagName, nil
}

func (u *Updater) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	return u.downloadWithProgress(ctx, url, limit, nil)
}

func (u *Updater) downloadWithProgress(
	ctx context.Context,
	url string,
	limit int64,
	progress func(downloaded, total int64),
) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "javaup-self-update/"+u.CurrentVersion)
	response, err := u.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("server returned %s", response.Status)
	}
	reader := io.Reader(io.LimitReader(response.Body, limit+1))
	if progress != nil {
		progress(0, response.ContentLength)
		reader = &downloadProgressReader{
			reader:     reader,
			total:      response.ContentLength,
			lastReport: time.Now(),
			report:     progress,
		}
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}

type downloadProgressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	lastReport time.Time
	report     func(downloaded, total int64)
}

func (r *downloadProgressReader) Read(buffer []byte) (int, error) {
	read, err := r.reader.Read(buffer)
	r.downloaded += int64(read)
	now := time.Now()
	if err == io.EOF || now.Sub(r.lastReport) >= 100*time.Millisecond {
		r.report(r.downloaded, r.total)
		r.lastReport = now
	}
	return read, err
}

type version struct {
	tag string
}

func parseVersion(value string) (version, error) {
	if !semver.IsValid(value) || semver.Canonical(value) != value ||
		semver.Prerelease(value) != "" || semver.Build(value) != "" ||
		strings.Count(value, ".") != 2 {
		return version{}, fmt.Errorf("invalid semantic version %q", value)
	}
	return version{tag: value}, nil
}

func compareVersions(left, right version) int {
	return semver.Compare(left.tag, right.tag)
}

func validatePlatform(goos, goarch string) error {
	if goos != "linux" && goos != "darwin" && goos != "windows" {
		return fmt.Errorf("unsupported operating system: %s", goos)
	}
	if goarch != "amd64" && goarch != "arm64" {
		return fmt.Errorf("unsupported architecture: %s", goarch)
	}
	return nil
}

func releaseArchiveName(tag, goos, goarch string) string {
	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}
	return "javaup-" + strings.TrimPrefix(tag, "v") + "-" + goos + "-" + goarch + extension
}

func expectedChecksum(contents []byte, archiveName string) (string, error) {
	for line := range strings.SplitSeq(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != archiveName {
			continue
		}
		checksum := strings.ToLower(fields[0])
		decoded, err := hex.DecodeString(checksum)
		if err == nil && len(decoded) == sha256.Size {
			return checksum, nil
		}
		break
	}
	return "", fmt.Errorf("checksum for %s was not found", archiveName)
}

func stageBinary(archive []byte, goos, target string) (string, error) {
	binaryName := "jup"
	if goos == "windows" {
		binaryName = "jup.exe"
	}

	var binary io.ReadCloser
	var err error
	if goos == "windows" {
		binary, err = binaryFromZip(archive, binaryName)
	} else {
		binary, err = binaryFromTarGzip(archive, binaryName)
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = binary.Close() }()

	stagedFile, err := os.CreateTemp(filepath.Dir(target), ".jup-update-*")
	if err != nil {
		return "", fmt.Errorf("create staged executable: %w", err)
	}
	staged := stagedFile.Name()
	clean := true
	defer func() {
		_ = stagedFile.Close()
		if clean {
			_ = os.Remove(staged)
		}
	}()

	written, err := io.Copy(stagedFile, io.LimitReader(binary, maxBinarySize+1))
	if err != nil {
		return "", fmt.Errorf("extract executable: %w", err)
	}
	if written > maxBinarySize {
		return "", errors.New("release executable is too large")
	}
	if err := stagedFile.Chmod(0o755); err != nil {
		return "", fmt.Errorf("make staged executable runnable: %w", err)
	}
	if err := stagedFile.Close(); err != nil {
		return "", fmt.Errorf("close staged executable: %w", err)
	}
	clean = false
	return staged, nil
}

func binaryFromZip(archive []byte, binaryName string) (io.ReadCloser, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open release archive: %w", err)
	}
	var match *zip.File
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() && filepath.Base(filepath.ToSlash(file.Name)) == binaryName {
			if match != nil {
				return nil, fmt.Errorf("release archive contains more than one %s", binaryName)
			}
			match = file
		}
	}
	if match == nil {
		return nil, fmt.Errorf("release archive does not contain %s", binaryName)
	}
	binary, err := match.Open()
	if err != nil {
		return nil, fmt.Errorf("open release executable: %w", err)
	}
	return binary, nil
}

func binaryFromTarGzip(archive []byte, binaryName string) (io.ReadCloser, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open release archive: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()
	tarReader := tar.NewReader(gzipReader)
	var contents []byte
	matches := 0
	for {
		header, nextErr := tarReader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("read release archive: %w", nextErr)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(filepath.ToSlash(header.Name)) != binaryName {
			continue
		}
		matches++
		if matches > 1 {
			return nil, fmt.Errorf("release archive contains more than one %s", binaryName)
		}
		contents, err = io.ReadAll(io.LimitReader(tarReader, maxBinarySize+1))
		if err != nil {
			return nil, fmt.Errorf("read release executable: %w", err)
		}
	}
	if matches == 0 {
		return nil, fmt.Errorf("release archive does not contain %s", binaryName)
	}
	return io.NopCloser(bytes.NewReader(contents)), nil
}
