// Package javainfo discovers installed Java development kits.
package javainfo

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var javaHomeVariablePattern = regexp.MustCompile(`(?i)^(?:JAVA_HOME|JDK_HOME|JAVA\d+_HOME|JAVA_HOME_\d+|JAVA_\d+_HOME|JDK\d+_HOME|JDK_HOME_\d+)$`)

// Installation identifies a locally installed JDK.
type Installation struct {
	Version string `json:"version"`
	Home    string `json:"home"`
}

// Locator searches common developer configuration and installation locations.
type Locator struct{}

// NewLocator creates a cross-platform JDK locator.
func NewLocator() *Locator {
	return &Locator{}
}

// Locate returns a JDK matching the requested major version. An empty version
// selects the first valid JDK, preferring build-tool runtime hints and JAVA_HOME.
func (l *Locator) Locate(ctx context.Context, version string, preferred ...Installation) (Installation, error) {
	requested, err := normalizeVersion(version)
	if err != nil {
		return Installation{}, err
	}

	candidates := make([]string, 0, 32)
	for _, installation := range preferred {
		if installation.Home != "" {
			candidates = append(candidates, installation.Home)
		}
	}
	candidates = append(candidates, environmentCandidates()...)
	if executable, err := exec.LookPath(javacExecutable()); err == nil {
		candidates = append(candidates, filepath.Dir(filepath.Dir(executable)))
	}
	candidates = append(candidates, siblingCandidates(candidates)...)
	candidates = append(candidates, toolchainCandidates()...)
	candidates = append(candidates, platformCandidates()...)

	seen := make(map[string]struct{})
	var discovered []Installation
	for _, candidate := range candidates {
		installation, ok := inspectCandidate(ctx, candidate)
		if !ok {
			continue
		}

		key := installation.Home
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		discovered = append(discovered, installation)

		if matchesVersion(installation.Version, requested) {
			return installation, nil
		}
	}

	if requested == "" {
		return Installation{}, fmt.Errorf("no installed JDK was found")
	}
	versions := make([]string, 0, len(discovered))
	for _, installation := range discovered {
		versions = append(versions, installation.Version+" at "+installation.Home)
	}
	if len(versions) == 0 {
		return Installation{}, fmt.Errorf("java %s is required but no installed JDK was found", requested)
	}
	return Installation{}, fmt.Errorf("java %s is required; discovered %s", requested, strings.Join(versions, ", "))
}

func environmentCandidates() []string {
	candidates := make([]string, 0, 8)
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		candidates = append(candidates, javaHome)
	}

	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if ok && value != "" && name != "JAVA_HOME" && javaHomeVariablePattern.MatchString(name) {
			candidates = append(candidates, value)
		}
	}
	return candidates
}

func siblingCandidates(candidates []string) []string {
	var siblings []string
	seenParents := make(map[string]struct{})
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(expandHome(candidate))
		if candidate == "" {
			continue
		}
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		absolute = filepath.Clean(absolute)
		parent := filepath.Dir(absolute)
		if parent == absolute {
			continue
		}

		key := parent
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, exists := seenParents[key]; exists {
			continue
		}
		seenParents[key] = struct{}{}

		entries, err := os.ReadDir(parent)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if (entry.IsDir() || entry.Type()&os.ModeSymlink != 0) && looksLikeJDKDirectory(entry.Name()) {
				siblings = append(siblings, filepath.Join(parent, entry.Name()))
			}
		}
	}
	return siblings
}

func looksLikeJDKDirectory(name string) bool {
	name = strings.ToLower(name)
	for _, marker := range []string{
		"java", "jdk", "temurin", "corretto", "zulu", "liberica", "graal", "semeru", "sapmachine",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func toolchainCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	toolchainsPath := filepath.Join(home, ".m2", "toolchains.xml")
	// #nosec G304 -- the path is the fixed Maven Toolchains location in the user's home directory.
	content, err := os.ReadFile(toolchainsPath)
	if err != nil {
		return nil
	}

	var document struct {
		Toolchains []struct {
			Type          string `xml:"type"`
			Configuration struct {
				JDKHome string `xml:"jdkHome"`
			} `xml:"configuration"`
		} `xml:"toolchain"`
	}
	if err := xml.Unmarshal(content, &document); err != nil {
		return nil
	}

	var candidates []string
	for _, toolchain := range document.Toolchains {
		if strings.EqualFold(strings.TrimSpace(toolchain.Type), "jdk") {
			candidates = append(candidates, expandHome(os.ExpandEnv(strings.TrimSpace(toolchain.Configuration.JDKHome))))
		}
	}
	return candidates
}

func platformCandidates() []string {
	home, _ := os.UserHomeDir()
	patterns := make([]string, 0, 12)

	switch runtime.GOOS {
	case "windows":
		programFiles := []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")}
		for _, root := range programFiles {
			if root == "" {
				continue
			}
			patterns = append(patterns,
				filepath.Join(root, "Java", "*"),
				filepath.Join(root, "Eclipse Adoptium", "*"),
				filepath.Join(root, "Microsoft", "jdk-*"),
				filepath.Join(root, "Amazon Corretto", "*"),
				filepath.Join(root, "BellSoft", "LibericaJDK-*"),
			)
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			patterns = append(patterns, filepath.Join(localAppData, "Programs", "Eclipse Adoptium", "*"))
		}
		if home != "" {
			patterns = append(patterns,
				filepath.Join(home, ".jdks", "*"),
				filepath.Join(home, "scoop", "apps", "*jdk*", "current"),
			)
		}
	case "darwin":
		patterns = append(patterns,
			filepath.Join(string(filepath.Separator), "Library", "Java", "JavaVirtualMachines", "*", "Contents", "Home"),
			filepath.Join(string(filepath.Separator), "opt", "homebrew", "opt", "openjdk*", "libexec", "openjdk.jdk", "Contents", "Home"),
		)
		if home != "" {
			patterns = append(patterns,
				filepath.Join(home, "Library", "Java", "JavaVirtualMachines", "*", "Contents", "Home"),
				filepath.Join(home, ".sdkman", "candidates", "java", "*"),
				filepath.Join(home, ".asdf", "installs", "java", "*"),
			)
		}
	default:
		patterns = append(patterns,
			filepath.Join(string(filepath.Separator), "usr", "lib", "jvm", "*"),
			filepath.Join(string(filepath.Separator), "usr", "java", "*"),
			filepath.Join(string(filepath.Separator), "opt", "java", "*"),
		)
		if home != "" {
			patterns = append(patterns,
				filepath.Join(home, ".sdkman", "candidates", "java", "*"),
				filepath.Join(home, ".asdf", "installs", "java", "*"),
				filepath.Join(home, ".jdks", "*"),
			)
		}
	}

	var candidates []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		candidates = append(candidates, matches...)
	}
	return candidates
}

func inspectCandidate(ctx context.Context, candidate string) (Installation, bool) {
	candidate = strings.TrimSpace(expandHome(candidate))
	if candidate == "" {
		return Installation{}, false
	}

	home, err := filepath.Abs(candidate)
	if err != nil {
		return Installation{}, false
	}
	if resolved, err := filepath.EvalSymlinks(home); err == nil {
		home = resolved
	}
	home = filepath.Clean(home)

	javac := filepath.Join(home, "bin", javacExecutable())
	if info, err := os.Stat(javac); err != nil || info.IsDir() {
		return Installation{}, false
	}

	version := releaseVersion(filepath.Join(home, "release"))
	if version == "" {
		// #nosec G204 -- javac is restricted to a validated JDK bin directory discovered locally.
		output, err := exec.CommandContext(ctx, javac, "-version").CombinedOutput()
		if err != nil {
			return Installation{}, false
		}
		fields := strings.Fields(string(output))
		if len(fields) >= 2 {
			version = validFullVersion(fields[1])
		}
	}
	if version == "" {
		return Installation{}, false
	}

	return Installation{Version: version, Home: home}, true
}

func releaseVersion(path string) string {
	// #nosec G304 -- path is always the release file under a validated JDK candidate.
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok && strings.TrimSpace(key) == "JAVA_VERSION" {
			return validFullVersion(strings.Trim(strings.TrimSpace(value), `"`))
		}
	}
	return ""
}

func validFullVersion(value string) string {
	value = strings.TrimSpace(value)
	if _, err := normalizeVersion(value); err != nil {
		return ""
	}
	return value
}

func matchesVersion(version, requested string) bool {
	if requested == "" {
		return true
	}
	major, err := normalizeVersion(version)
	return err == nil && major == requested
}

func normalizeVersion(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	value = strings.TrimPrefix(value, "1.")
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return "", fmt.Errorf("unsupported Java version %q", value)
	}
	return value[:end], nil
}

func javacExecutable() string {
	if runtime.GOOS == "windows" {
		return "javac.exe"
	}
	return "javac"
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimLeft(path[1:], `/\`))
		}
	}
	return path
}
