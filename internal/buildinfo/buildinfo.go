// Package buildinfo exposes values that can be replaced at build time.
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

const shortCommitLength = 12

var (
	// Version is the semantic version reported by the application.
	Version = "v0.2.0"
	// Commit optionally overrides the revision embedded by the Go toolchain.
	Commit string
)

// Info contains immutable build metadata displayed by the CLI.
type Info struct {
	Version  string
	Platform string
	Commit   string
}

// Current returns build metadata for the running binary.
func Current() Info {
	commit := Commit
	if commit == "" {
		commit = vcsRevision()
	}

	return newInfo(Version, runtime.GOOS, runtime.GOARCH, commit)
}

func newInfo(version, goos, goarch, commit string) Info {
	if commit == "" {
		commit = "unknown"
	}
	if len(commit) > shortCommitLength {
		commit = commit[:shortCommitLength]
	}

	return Info{
		Version:  version,
		Platform: goos + "/" + goarch,
		Commit:   commit,
	}
}

func vcsRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}

	return ""
}
