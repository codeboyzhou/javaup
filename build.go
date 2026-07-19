// Command build runs the complete local verification and build pipeline.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

type buildStep struct {
	name          string
	description   string
	program       string
	args          []string
	requireSilent bool
}

type palette struct {
	info    *color.Color
	error   *color.Color
	heading *color.Color
	stage   *color.Color
	command *color.Color
	success *color.Color
	failure *color.Color
}

type buildResult struct {
	artifact   string
	failedStep string
}

const separator = "------------------------------------------------------------------------"

func main() {
	colors := newPalette()
	started := time.Now()
	verifyOnly, err := parseBuildMode(os.Args[1:])
	if err != nil {
		printSummary(colors, started, verifyOnly, buildResult{}, err)
		printUsage(os.Stderr)
		os.Exit(2)
	}

	result, err := runBuild(colors, verifyOnly)
	printSummary(colors, started, verifyOnly, result, err)
	if err != nil {
		os.Exit(1)
	}
}

func parseBuildMode(arguments []string) (verifyOnly bool, err error) {
	if len(arguments) == 0 {
		return false, nil
	}
	if len(arguments) == 1 && arguments[0] == "verify" {
		return true, nil
	}
	return false, fmt.Errorf("invalid build arguments: %s", strings.Join(arguments, " "))
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `Usage:
  go run build.go
  go run build.go verify`)
}

func runBuild(colors palette, verifyOnly bool) (buildResult, error) {
	if _, err := os.Stat("go.mod"); err != nil {
		return buildResult{}, fmt.Errorf("run this script from the repository root: %w", err)
	}

	steps := verificationSteps()
	if verifyOnly {
		printInfo(colors, colors.heading, separator)
		printInfo(colors, colors.heading, "VERIFYING JUP")
		printInfo(colors, colors.heading, separator)
		return executeSteps(colors, steps, buildResult{})
	}

	goos, goarch, err := targetPlatform()
	if err != nil {
		return buildResult{}, err
	}

	artifact := filepath.Join("dist", binaryName(goos))
	result := buildResult{artifact: artifact}
	if err := os.MkdirAll(filepath.Dir(artifact), 0o750); err != nil {
		return result, fmt.Errorf("create output directory: %w", err)
	}

	steps = append(steps, buildStep{
		name:        "Build",
		description: "Building the jup executable",
		program:     "go",
		args:        []string{"build", "-trimpath", "-o", artifact, "./cmd/jup"},
	})

	printInfo(colors, colors.heading, separator)
	printInfo(colors, colors.heading, "BUILDING JUP")
	printInfo(colors, colors.heading, separator)
	printInfo(colors, nil, "Target:   %s", goos+"/"+goarch)
	printInfo(colors, nil, "Artifact: %s", artifact)

	return executeSteps(colors, steps, result)
}

func verificationSteps() []buildStep {
	return []buildStep{
		{
			name:          "Format",
			description:   "Checking Go source formatting",
			program:       "gofmt",
			args:          []string{"-l", "."},
			requireSilent: true,
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
	}
}

func executeSteps(colors palette, steps []buildStep, result buildResult) (buildResult, error) {
	for index, step := range steps {
		printInfo(colors, nil, "")
		if err := runStep(colors, index+1, len(steps), step); err != nil {
			result.failedStep = step.name
			return result, err
		}
	}

	return result, nil
}

func runStep(colors palette, index, total int, step buildStep) error {
	stage := fmt.Sprintf("--- %s (%d/%d) ---", strings.ToLower(step.name), index, total)
	printInfo(colors, colors.stage, "%s", stage)
	printInfo(colors, nil, "%s", step.description)
	commandLine := "$ " + strings.Join(append([]string{step.program}, step.args...), " ")
	printInfo(colors, colors.command, "%s", commandLine)

	// #nosec G204 -- every command and argument is defined by the build script.
	command := exec.Command(step.program, step.args...)
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	if step.requireSilent {
		output, err := command.Output()
		if err != nil {
			return fmt.Errorf("%s step: %w", strings.ToLower(step.name), err)
		}
		if output := strings.TrimSpace(string(output)); output != "" {
			return fmt.Errorf("%s step: files require formatting:\n%s", strings.ToLower(step.name), output)
		}
		return nil
	}
	command.Stdout = os.Stdout

	if err := command.Run(); err != nil {
		return fmt.Errorf("%s step: %w", strings.ToLower(step.name), err)
	}
	return nil
}

func printSummary(colors palette, started time.Time, verifyOnly bool, result buildResult, buildErr error) {
	writer := io.Writer(os.Stdout)
	prefix := colors.info
	status := colors.success
	operation := "BUILD"
	if verifyOnly {
		operation = "VERIFICATION"
	}
	outcome := operation + " SUCCESS"
	if buildErr != nil {
		writer = os.Stderr
		prefix = colors.error
		status = colors.failure
		outcome = operation + " FAILURE"
	}

	printLog(writer, colors, prefix, status, separator)
	printLog(writer, colors, prefix, status, "%s", outcome)
	printLog(writer, colors, prefix, status, separator)
	if result.failedStep != "" {
		printLog(writer, colors, prefix, nil, "Failed at:   %s", result.failedStep)
	}
	if buildErr != nil {
		printLog(writer, colors, prefix, nil, "Reason:      %v", buildErr)
	}
	printLog(writer, colors, prefix, nil, "Total time:  %s", formatDuration(time.Since(started)))
	printLog(writer, colors, prefix, nil, "Finished at: %s", time.Now().Format("2006-01-02T15:04:05Z07:00"))
	if buildErr == nil && result.artifact != "" {
		printLog(writer, colors, prefix, nil, "Artifact:    %s", result.artifact)
	}
	printLog(writer, colors, prefix, status, separator)
}

func printInfo(colors palette, style *color.Color, format string, args ...any) {
	printLog(os.Stdout, colors, colors.info, style, format, args...)
}

func printLog(
	writer io.Writer,
	colors palette,
	prefixStyle *color.Color,
	messageStyle *color.Color,
	format string,
	args ...any,
) {
	prefix := colors.apply(prefixStyle, "[INFO]")
	if prefixStyle == colors.error {
		prefix = colors.apply(prefixStyle, "[ERROR]")
	}
	message := fmt.Sprintf(format, args...)
	if messageStyle != nil {
		message = colors.apply(messageStyle, message)
	}
	_, _ = fmt.Fprintf(writer, "%s %s\n", prefix, message)
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
		info:    style(color.Bold, color.FgBlue),
		error:   style(color.Bold, color.FgRed),
		heading: style(color.Bold, color.FgGreen),
		stage:   style(color.Bold, color.FgGreen),
		command: style(color.Faint, color.FgWhite),
		success: style(color.Bold, color.FgGreen),
		failure: style(color.Bold, color.FgRed),
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

func formatDuration(duration time.Duration) string {
	return fmt.Sprintf("%.3f s", duration.Round(time.Millisecond).Seconds())
}
