package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBuildMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		arguments  []string
		wantVerify bool
		wantError  bool
	}{
		{name: "development"},
		{name: "verification", arguments: []string{"verify"}, wantVerify: true},
		{name: "unknown", arguments: []string{"release"}, wantError: true},
		{name: "extra argument", arguments: []string{"verify", "unexpected"}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			verifyOnly, err := parseBuildMode(test.arguments)
			if (err != nil) != test.wantError {
				t.Fatalf("parseBuildMode() error = %v, wantError = %t", err, test.wantError)
			}
			if verifyOnly != test.wantVerify {
				t.Errorf("parseBuildMode() verifyOnly = %t, want %t", verifyOnly, test.wantVerify)
			}
		})
	}
}

func TestVerificationStepsDoNotBuildBinary(t *testing.T) {
	t.Parallel()

	steps := verificationSteps()
	for _, step := range steps {
		if step.name == "Build" || step.program == "go" && len(step.args) > 0 && step.args[0] == "build" {
			t.Errorf("verificationSteps() contains build step %#v", step)
		}
	}
}

func TestVerificationFormatStepDoesNotModifyFiles(t *testing.T) {
	t.Parallel()

	for _, step := range verificationSteps() {
		if step.name != "Format" {
			continue
		}
		if step.program != "gofmt" || len(step.args) != 2 || step.args[0] != "-l" || step.args[1] != "." {
			t.Errorf("Format step command = %q %#v, want gofmt -l .", step.program, step.args)
		}
		if !step.requireSilent {
			t.Error("Format step requireSilent = false, want formatting differences to fail verification")
		}
		return
	}
	t.Fatal("verificationSteps() has no Format step")
}

func TestRunStepRejectsOutputFromSilentCommand(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "unformatted.go")
	if err := os.WriteFile(path, []byte("package sample\nfunc example( ){ }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	step := buildStep{
		name:          "Format",
		program:       "gofmt",
		args:          []string{"-l", path},
		requireSilent: true,
	}
	if err := runStep(newPalette(), 1, 1, step); err == nil || !strings.Contains(err.Error(), "files require formatting") {
		t.Fatalf("runStep() error = %v, want formatting failure", err)
	}
}
