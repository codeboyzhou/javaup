// Package buildtool defines abstractions shared by project build tool detectors.
package buildtool

import "context"

// Type identifies a supported project build tool.
type Type string

const (
	// Maven identifies Apache Maven projects.
	Maven Type = "maven"
)

// DisplayName returns the user-facing name of a build tool.
func (t Type) DisplayName() string {
	switch t {
	case Maven:
		return "Maven"
	default:
		return string(t)
	}
}

// Info contains the detected build tool configuration.
type Info struct {
	Type       Type   `json:"type"`
	Version    string `json:"version"`
	Executable string `json:"executable"`
	Wrapper    bool   `json:"wrapper"`
}

// JavaRuntime describes the Java runtime used to launch a build tool.
type JavaRuntime struct {
	Version string
	Home    string
}

// Detection is the normalized result returned by a build tool detector.
type Detection struct {
	Tool             Info
	BuildJavaVersion string
	RuntimeJava      JavaRuntime
}

// Detector discovers one kind of build tool in a project root.
type Detector interface {
	Detect(ctx context.Context, root string) (detection Detection, found bool, err error)
}
