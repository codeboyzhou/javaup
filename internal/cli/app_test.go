package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
			wantOutput: []string{"Usage:\n  jup [flags]", "help", "version"},
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
			wantOutput: []string{"jup v1.2.3"},
		},
		{
			name:       "version flag prints version",
			args:       []string{"--version"},
			wantCode:   exitSuccess,
			wantOutput: []string{"jup v1.2.3"},
		},
		{
			name:       "short version flag prints version",
			args:       []string{"-v"},
			wantCode:   exitSuccess,
			wantOutput: []string{"jup v1.2.3"},
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
				Description: "A command-line tool for managing Java versions.",
				Version:     "v1.2.3",
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

func assertContains(t *testing.T, got string, wants []string) {
	t.Helper()

	got = strings.ReplaceAll(got, "\r\n", "\n")
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output %q does not contain %q", got, want)
		}
	}
}
