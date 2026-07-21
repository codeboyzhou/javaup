package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAppRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput []string
		wantError  []string
	}{
		{
			name:       "no arguments shows help",
			wantCode:   exitSuccess,
			wantOutput: []string{"Usage:\n  jup [flags]", "help", "run", "status", "uninstall", "update", "version"},
		},
		{
			name:       "help command shows help",
			args:       []string{"help"},
			wantCode:   exitSuccess,
			wantOutput: []string{"Available Commands:", `Use "jup [command] --help"`},
		},
		{
			name:       "help flag shows help",
			args:       []string{"--help"},
			wantCode:   exitSuccess,
			wantOutput: []string{"Available Commands:"},
		},
		{
			name:       "command help shows command usage",
			args:       []string{"help", "version"},
			wantCode:   exitSuccess,
			wantOutput: []string{"Print version information", "Usage:\n  jup version [flags]"},
		},
		{
			name:       "version command prints version",
			args:       []string{"version"},
			wantCode:   exitSuccess,
			wantOutput: []string{"javaup version v1.2.3 testos/testarch (0123456789ab)"},
		},
		{
			name:       "version flag prints version",
			args:       []string{"--version"},
			wantCode:   exitSuccess,
			wantOutput: []string{"javaup version v1.2.3 testos/testarch (0123456789ab)"},
		},
		{
			name:       "short version flag prints version",
			args:       []string{"-v"},
			wantCode:   exitSuccess,
			wantOutput: []string{"javaup version v1.2.3 testos/testarch (0123456789ab)"},
		},
		{
			name:      "unknown command fails",
			args:      []string{"missing"},
			wantCode:  exitFailure,
			wantError: []string{`jup: unknown command "missing" for "jup"`},
		},
		{
			name:      "version rejects arguments",
			args:      []string{"version", "extra"},
			wantCode:  exitFailure,
			wantError: []string{"jup: unknown command \"extra\" for \"jup version\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			app := New(Options{
				Name:        "jup",
				ProductName: "javaup",
				Description: "A command-line tool for managing Java versions.",
				Version:     "v1.2.3",
				Platform:    "testos/testarch",
				Commit:      "0123456789ab",
				Stdout:      &stdout,
				Stderr:      &stderr,
			})

			if got := app.Run(context.Background(), tt.args); got != tt.wantCode {
				t.Fatalf("Run() exit code = %d, want %d", got, tt.wantCode)
			}
			assertContains(t, stdout.String(), tt.wantOutput)
			assertContains(t, stderr.String(), tt.wantError)
		})
	}
}

type testExitError struct {
	code int
}

func (e testExitError) Error() string {
	return "command failed"
}

func (e testExitError) ExitCode() int {
	return e.code
}

func TestAppReturnsChildProcessExitCodeWithoutDuplicateError(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	root := newRootCommand(Options{
		Name:        "jup",
		ProductName: "javaup",
		Description: "test",
		Stdout:      &bytes.Buffer{},
		Stderr:      &stderr,
	})
	root.AddCommand(&cobra.Command{
		Use: "failure",
		RunE: func(_ *cobra.Command, _ []string) error {
			return testExitError{code: 23}
		},
	})
	app := &App{root: root, stderr: &stderr}

	if got := app.Run(context.Background(), []string{"failure"}); got != 23 {
		t.Errorf("Run() exit code = %d, want 23", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("Run() stderr = %q, want empty", stderr.String())
	}
}

func TestVersionOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	app := New(Options{
		Name:        "jup",
		ProductName: "javaup",
		Description: "A command-line tool for managing Java versions.",
		Version:     "v0.1.0",
		Platform:    "windows/amd64",
		Commit:      "64c2fb07bcad",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
	})

	if got := app.Run(context.Background(), []string{"version"}); got != exitSuccess {
		t.Fatalf("Run() exit code = %d, want %d", got, exitSuccess)
	}
	const want = "javaup version v0.1.0 windows/amd64 (64c2fb07bcad)\n"
	if got := normalizedOutput(stdout.String()); got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}
}

func assertContains(t *testing.T, got string, wants []string) {
	t.Helper()

	got = normalizedOutput(got)
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output %q does not contain %q", got, want)
		}
	}
}

func normalizedOutput(output string) string {
	return strings.ReplaceAll(output, "\r\n", "\n")
}
