// Command build runs the complete local verification and build pipeline.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

type buildStep struct {
	name        string
	description string
	program     string
	args        []string
}

type palette struct {
	title    *color.Color
	label    *color.Color
	value    *color.Color
	stage    *color.Color
	command  *color.Color
	success  *color.Color
	failure  *color.Color
	duration *color.Color
}

func main() {
	colors := newPalette()
	if err := runBuild(colors); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", colors.apply(colors.failure, "BUILD FAILED"), err)
		os.Exit(1)
	}
}

func runBuild(colors palette) error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run this script from the repository root: %w", err)
	}

	goos, goarch, err := targetPlatform()
	if err != nil {
		return err
	}

	artifact := filepath.Join("dist", binaryName(goos))
	if err := os.MkdirAll(filepath.Dir(artifact), 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	steps := []buildStep{
		{
			name:        "Format",
			description: "Formatting Go source files",
			program:     "go",
			args:        []string{"fmt", "./..."},
		},
		{
			name:        "Vet",
			description: "Running Go static analysis",
			program:     "go",
			args:        []string{"vet", "./..."},
		},
		{
			name:        "Lint",
			description: "Running GolangCI-Lint",
			program:     "go",
			args:        []string{"tool", "-modfile=golangci-lint.mod", "golangci-lint", "run"},
		},
		{
			name:        "Test",
			description: "Running all tests",
			program:     "go",
			args:        []string{"test", "./..."},
		},
		{
			name:        "Vulncheck",
			description: "Scanning dependencies for known vulnerabilities",
			program:     "go",
			args:        []string{"tool", "-modfile=govulncheck.mod", "govulncheck", "./..."},
		},
		{
			name:        "Build",
			description: "Building the jup executable",
			program:     "go",
			args:        []string{"build", "-trimpath", "-o", artifact, "./cmd/jup"},
		},
	}

	started := time.Now()
	fmt.Printf(
		"%s | %s %s | %s %s\n",
		colors.apply(colors.title, "JUP BUILD"),
		colors.apply(colors.label, "Target:"),
		colors.apply(colors.value, goos+"/"+goarch),
		colors.apply(colors.label, "Artifact:"),
		colors.apply(colors.value, artifact),
	)

	for index, step := range steps {
		if err := runStep(colors, index+1, len(steps), step); err != nil {
			return err
		}
	}

	fmt.Printf(
		"%s %s | %s %s\n",
		colors.apply(colors.success, "BUILD SUCCEEDED"),
		colors.apply(colors.duration, "in "+elapsed(started).String()),
		colors.apply(colors.label, "Artifact:"),
		colors.apply(colors.value, artifact),
	)
	return nil
}

func runStep(colors palette, index, total int, step buildStep) error {
	stage := fmt.Sprintf("[%d/%d] %s", index, total, step.name)
	fmt.Printf("%s - %s\n", colors.apply(colors.stage, stage), step.description)
	commandLine := "$ " + strings.Join(append([]string{step.program}, step.args...), " ")
	fmt.Printf("      %s\n", colors.apply(colors.command, commandLine))

	started := time.Now()
	// #nosec G204 -- every command and argument is defined by the build script.
	command := exec.Command(step.program, step.args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		result := fmt.Sprintf("[%d/%d] %s Failed", index, total, step.name)
		fmt.Printf(
			"%s %s\n",
			colors.apply(colors.failure, result),
			colors.apply(colors.duration, "in "+elapsed(started).String()),
		)
		return fmt.Errorf("%s step: %w", strings.ToLower(step.name), err)
	}

	result := fmt.Sprintf("[%d/%d] %s OK", index, total, step.name)
	fmt.Printf(
		"%s %s\n",
		colors.apply(colors.success, result),
		colors.apply(colors.duration, "in "+elapsed(started).String()),
	)
	return nil
}

func newPalette() palette {
	enabled := colorsEnabled()
	style := func(attributes ...color.Attribute) *color.Color {
		value := color.New(attributes...)
		if enabled {
			value.EnableColor()
		} else {
			value.DisableColor()
		}
		return value
	}

	return palette{
		title:    style(color.FgCyan),
		label:    style(color.FgCyan),
		value:    style(),
		stage:    style(color.FgBlue),
		command:  style(color.Faint, color.FgWhite),
		success:  style(color.FgGreen),
		failure:  style(color.FgRed),
		duration: style(color.FgYellow),
	}
}

func colorsEnabled() bool {
	switch strings.ToLower(os.Getenv("JUP_BUILD_COLOR")) {
	case "always":
		return true
	case "never":
		return false
	}

	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}

	return !color.NoColor
}

func (p palette) apply(style *color.Color, text string) string {
	return style.Sprint(text)
}

func targetPlatform() (string, string, error) {
	output, err := exec.Command("go", "env", "GOOS", "GOARCH").Output()
	if err != nil {
		return "", "", fmt.Errorf("read Go target platform: %w", err)
	}

	values := strings.Fields(string(output))
	if len(values) != 2 {
		return "", "", fmt.Errorf("read Go target platform: unexpected output %q", output)
	}

	return values[0], values[1], nil
}

func binaryName(goos string) string {
	if goos == "windows" {
		return "jup.exe"
	}
	return "jup"
}

func elapsed(started time.Time) time.Duration {
	return time.Since(started).Round(time.Millisecond)
}
