package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/selfupdate"
)

type stubUpdateService struct {
	checkResult  selfupdate.Result
	updateResult selfupdate.Result
	err          error
}

func (s stubUpdateService) Check(context.Context) (selfupdate.Result, error) {
	return s.checkResult, s.err
}

func (s stubUpdateService) UpdateWithProgress(
	_ context.Context,
	progress selfupdate.ProgressFunc,
) (selfupdate.Result, error) {
	progress(selfupdate.ProgressEvent{Stage: selfupdate.ProgressChecking})
	if s.updateResult.Updated {
		progress(selfupdate.ProgressEvent{
			Stage:   selfupdate.ProgressAvailable,
			Current: s.updateResult.Current,
			Latest:  s.updateResult.Latest,
		})
		progress(selfupdate.ProgressEvent{
			Stage: selfupdate.ProgressFetchingChecksums,
			URL:   "https://example.com/download/v1.1.0/checksums.txt",
		})
		progress(selfupdate.ProgressEvent{
			Stage: selfupdate.ProgressConnecting,
			Name:  "javaup-1.1.0-windows-amd64.zip",
			URL:   "https://example.com/download/v1.1.0/javaup-1.1.0-windows-amd64.zip",
		})
		progress(selfupdate.ProgressEvent{
			Stage: selfupdate.ProgressDownloading,
			Name:  "javaup-1.1.0-windows-amd64.zip",
			Total: 2048,
		})
		progress(selfupdate.ProgressEvent{
			Stage:      selfupdate.ProgressDownloading,
			Name:       "javaup-1.1.0-windows-amd64.zip",
			Downloaded: 2048,
			Total:      2048,
		})
		progress(selfupdate.ProgressEvent{Stage: selfupdate.ProgressVerifying})
		progress(selfupdate.ProgressEvent{Stage: selfupdate.ProgressExtracting})
		progress(selfupdate.ProgressEvent{Stage: selfupdate.ProgressInstalling})
	}
	return s.updateResult, s.err
}

func TestUpdateCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		service stubUpdateService
		want    string
		wantErr string
	}{
		{
			name: "checks for an available update",
			args: []string{"--check"},
			service: stubUpdateService{checkResult: selfupdate.Result{
				Current: "v1.0.0", Latest: "v1.1.0", Updated: true,
			}},
			want: "Update available: v1.0.0 -> v1.1.0",
		},
		{
			name: "reports current version",
			service: stubUpdateService{updateResult: selfupdate.Result{
				Current: "v1.1.0", Latest: "v1.1.0",
			}},
			want: "Already up to date (v1.1.0)",
		},
		{
			name: "reports completed update",
			service: stubUpdateService{updateResult: selfupdate.Result{
				Current: "v1.0.0", Latest: "v1.1.0", Updated: true,
			}},
			want: "Updated successfully",
		},
		{
			name: "reports pending Windows update as successful",
			service: stubUpdateService{updateResult: selfupdate.Result{
				Current: "v1.0.0", Latest: "v1.1.0", Updated: true, Pending: true,
			}},
			want: "Updated successfully",
		},
		{
			name: "reports scoop-style update progress",
			service: stubUpdateService{updateResult: selfupdate.Result{
				Current: "v1.0.0", Latest: "v1.1.0", Updated: true,
			}},
			want: strings.Join([]string{
				"Checking for updates...",
				"Updating jup (v1.0.0 -> v1.1.0)",
				"Fetching checksums from https://example.com/download/v1.1.0/checksums.txt",
				"Downloading new version from https://example.com/download/v1.1.0/javaup-1.1.0-windows-amd64.zip",
				"javaup-1.1.0-windows-amd64.zip (2.0 KB)...",
				"Checking hash... ok",
				"Extracting archive...",
				"Installing update...",
			}, "\n"),
		},
		{
			name:    "returns update errors",
			service: stubUpdateService{err: errors.New("network unavailable")},
			wantErr: "network unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			command := newUpdateCommand(func() updateService { return tt.service })
			command.SetOut(&output)
			command.SetArgs(tt.args)
			err := command.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(output.String(), tt.want) {
				t.Errorf("output = %q, want %q", output.String(), tt.want)
			}
		})
	}
}

func TestFormatDownloadProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		downloaded int64
		total      int64
		want       string
	}{
		{
			name:       "known size",
			downloaded: 1024,
			total:      4096,
			want:       "archive.zip (4.0 KB) [========>                       ]  25%",
		},
		{
			name:       "complete",
			downloaded: 4096,
			total:      4096,
			want:       "archive.zip (4.0 KB) [================================] 100%",
		},
		{
			name:       "unknown size",
			downloaded: 1536,
			total:      -1,
			want:       "archive.zip (1.5 KB downloaded)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDownloadProgress("archive.zip", tt.downloaded, tt.total, 60)
			if got != tt.want {
				t.Errorf("formatDownloadProgress() = %q, want %q", got, tt.want)
			}
			if tt.total > 0 && len(got) != 60 {
				t.Errorf("progress width = %d, want 60", len(got))
			}
		})
	}
}

func TestUpdateProgressRendererRefreshesInteractiveDownloadLine(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	renderer := newUpdateProgressRenderer(&output)
	renderer.interactive = true
	renderer.terminalWidth = func() int { return 60 }
	renderer.Report(selfupdate.ProgressEvent{
		Stage: selfupdate.ProgressConnecting, Name: "archive.zip", URL: "https://example.com/archive.zip",
	})
	renderer.Report(selfupdate.ProgressEvent{
		Stage: selfupdate.ProgressDownloading, Name: "archive.zip", Downloaded: 1024, Total: 4096,
	})
	renderer.Report(selfupdate.ProgressEvent{
		Stage: selfupdate.ProgressDownloading, Name: "archive.zip", Downloaded: 4096, Total: 4096,
	})
	renderer.Report(selfupdate.ProgressEvent{Stage: selfupdate.ProgressVerifying})
	if err := renderer.Finish(); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}

	want := strings.Join([]string{
		"Downloading new version from https://example.com/archive.zip\n",
		"\rarchive.zip (4.0 KB) [========>                       ]  25%",
		"\rarchive.zip (4.0 KB) [================================] 100%",
		"\nChecking hash... ok\n",
	}, "")
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}
