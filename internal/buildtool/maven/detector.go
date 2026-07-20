// Package maven detects Apache Maven project configuration.
package maven

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/codeboyzhou/javaup/internal/buildtool"
)

var (
	mavenOutputVersionPattern = regexp.MustCompile(`(?m)^Apache Maven\s+([^\s]+)`)
	mavenOutputJavaPattern    = regexp.MustCompile(`(?m)^Java version:\s*([^,\r\n]+).*runtime:\s*(.+?)\s*$`)
	wrapperPathVersionPattern = regexp.MustCompile(`/apache-maven/([^/]+)/apache-maven-`)
	wrapperFileVersionPattern = regexp.MustCompile(`apache-maven-([0-9][0-9A-Za-z_.-]*)-bin\.(?:zip|tar\.gz)`)
)

type commandRunner interface {
	Run(ctx context.Context, root, executable string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, root, executable string) ([]byte, error) {
	command := platformMavenVersionCommand(ctx, executable)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			if filepath.IsAbs(executable) {
				return nil, fmt.Errorf("maven executable does not exist: %s", executable)
			}
			return nil, fmt.Errorf(
				"maven executable %q was not found in PATH; install Maven, add mvn to PATH, or add Maven Wrapper to the project",
				executable,
			)
		}
		return nil, fmt.Errorf("run %s --version: %w: %s", executable, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

// Detector discovers Maven projects without invoking a wrapper when its
// distribution URL already identifies the configured Maven version.
type Detector struct {
	runner commandRunner
}

// NewDetector creates a Maven detector backed by local command execution.
func NewDetector() *Detector {
	return &Detector{runner: execRunner{}}
}

// Detect implements buildtool.Detector.
func (d *Detector) Detect(ctx context.Context, root string) (buildtool.Detection, bool, error) {
	pomPath := filepath.Join(root, "pom.xml")
	if _, err := os.Stat(pomPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return buildtool.Detection{}, false, nil
		}
		return buildtool.Detection{}, false, fmt.Errorf("inspect Maven POM: %w", err)
	}

	javaVersion, err := detectBuildJavaVersion(pomPath)
	if err != nil {
		return buildtool.Detection{}, true, err
	}

	wrapperExecutable, hasWrapper := findWrapper(root)
	executable := "mvn"
	if hasWrapper {
		executable = wrapperExecutable
	}
	executable = resolveExecutable(executable)
	version := ""
	runtimeJava := buildtool.JavaRuntime{}
	if hasWrapper {
		version, err = wrapperMavenVersion(root)
		if err != nil {
			return buildtool.Detection{}, true, err
		}
	}

	if version == "" {
		output, runErr := d.runner.Run(ctx, root, executable)
		if runErr != nil {
			return buildtool.Detection{}, true, runErr
		}
		version, runtimeJava, err = parseMavenVersionOutput(output)
		if err != nil {
			return buildtool.Detection{}, true, err
		}
	}

	return buildtool.Detection{
		Tool: buildtool.Info{
			Type:       buildtool.Maven,
			Version:    version,
			Executable: executable,
			Wrapper:    hasWrapper,
		},
		BuildJavaVersion: javaVersion,
		RuntimeJava:      runtimeJava,
	}, true, nil
}

func resolveExecutable(executable string) string {
	resolved, err := exec.LookPath(executable)
	if err != nil {
		return filepath.Clean(executable)
	}
	absolute, err := filepath.Abs(resolved)
	if err == nil {
		resolved = absolute
	}
	if evaluated, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = evaluated
	}
	return filepath.Clean(resolved)
}

func findWrapper(root string) (string, bool) {
	candidates := []string{"mvnw", "mvnw.cmd"}
	if runtime.GOOS == "windows" {
		candidates[0], candidates[1] = candidates[1], candidates[0]
	}

	for _, candidate := range candidates {
		path := filepath.Join(root, candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func wrapperMavenVersion(root string) (string, error) {
	propertiesPath := filepath.Join(root, ".mvn", "wrapper", "maven-wrapper.properties")
	// #nosec G304 -- the path is a fixed Maven Wrapper location under the detected project root.
	file, err := os.Open(propertiesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read Maven wrapper properties: %w", err)
	}
	defer func() { _ = file.Close() }()

	properties := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			key, value, ok = strings.Cut(line, ":")
		}
		if ok {
			properties[strings.TrimSpace(key)] = unescapeProperty(strings.TrimSpace(value))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan Maven wrapper properties: %w", err)
	}

	distributionURL := strings.ReplaceAll(properties["distributionUrl"], "\\", "/")
	if match := wrapperPathVersionPattern.FindStringSubmatch(distributionURL); len(match) == 2 {
		return match[1], nil
	}
	if match := wrapperFileVersionPattern.FindStringSubmatch(distributionURL); len(match) == 2 {
		return match[1], nil
	}
	return "", nil
}

func unescapeProperty(value string) string {
	value = strings.ReplaceAll(value, `\:`, ":")
	value = strings.ReplaceAll(value, `\=`, "=")
	return value
}

func parseMavenVersionOutput(output []byte) (string, buildtool.JavaRuntime, error) {
	versionMatch := mavenOutputVersionPattern.FindSubmatch(output)
	if len(versionMatch) != 2 {
		return "", buildtool.JavaRuntime{}, fmt.Errorf("maven version output does not contain an Apache Maven version")
	}

	runtimeJava := buildtool.JavaRuntime{}
	if javaMatch := mavenOutputJavaPattern.FindSubmatch(output); len(javaMatch) == 3 {
		version, err := normalizeJavaVersion(string(javaMatch[1]))
		if err == nil {
			runtimeJava.Version = version
		}
		runtimeJava.Home = filepath.Clean(strings.TrimSpace(string(javaMatch[2])))
	}

	return strings.TrimSpace(string(versionMatch[1])), runtimeJava, nil
}
