package project

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/buildtool/maven"
	"github.com/codeboyzhou/javaup/internal/javainfo"
)

type javaLocator interface {
	Locate(ctx context.Context, version string, preferred ...javainfo.Installation) (javainfo.Installation, error)
}

type configStore interface {
	Save(config Config) (string, error)
}

// Initializer detects a project's build environment and persists it locally.
type Initializer struct {
	detectors []buildtool.Detector
	java      javaLocator
	store     configStore
	now       func() time.Time
}

const initializationSteps = 5

// NewDefaultInitializer wires all currently supported project detectors.
func NewDefaultInitializer() (*Initializer, error) {
	store, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	return NewInitializer(
		[]buildtool.Detector{maven.NewDetector()},
		javainfo.NewLocator(),
		store,
	), nil
}

// NewInitializer creates an initializer from replaceable detection services.
func NewInitializer(detectors []buildtool.Detector, java javaLocator, store configStore) *Initializer {
	return &Initializer{
		detectors: detectors,
		java:      java,
		store:     store,
		now:       time.Now,
	}
}

// Initialize scans root, reports meaningful progress, and writes its project configuration.
func (i *Initializer) Initialize(ctx context.Context, root string, progress ProgressFunc) (Config, string, error) {
	reportProgress(progress, ProgressEvent{
		Step: 1, Total: initializationSteps, Name: "PROJECT", State: ProgressStarted,
		Message: "Inspect current project directory",
	})
	canonicalRoot, err := canonicalProjectRoot(root)
	if err != nil {
		reportFailure(progress, 1, "PROJECT", err)
		return Config{}, "", err
	}
	reportSuccess(progress, 1, "PROJECT", canonicalRoot)

	reportProgress(progress, ProgressEvent{
		Step: 2, Total: initializationSteps, Name: "BUILD TOOL", State: ProgressStarted,
		Message: "Detect build tool, version, and wrapper",
	})
	var detection buildtool.Detection
	found := false
	for _, detector := range i.detectors {
		detection, found, err = detector.Detect(ctx, canonicalRoot)
		if err != nil {
			reportFailure(progress, 2, "BUILD TOOL", err)
			return Config{}, "", err
		}
		if found {
			break
		}
	}
	if !found {
		err := fmt.Errorf("no supported build project found in %s", canonicalRoot)
		reportFailure(progress, 2, "BUILD TOOL", err)
		return Config{}, "", err
	}
	tool := string(detection.Tool.Type) + " " + detection.Tool.Version
	reportProgress(progress, ProgressEvent{
		Step: 2, Total: initializationSteps, Name: "BUILD TOOL", State: ProgressInfo,
		Message: wrapperProgressMessage(detection.Tool),
	})
	reportSuccess(progress, 2, "BUILD TOOL", tool)

	reportProgress(progress, ProgressEvent{
		Step: 3, Total: initializationSteps, Name: "JAVA VERSION", State: ProgressStarted,
		Message: "Detect configured Java build version",
	})
	if detection.BuildJavaVersion == "" {
		reportSuccess(progress, 3, "JAVA VERSION", "Use the build runtime JDK")
	} else {
		reportSuccess(progress, 3, "JAVA VERSION", "Java "+detection.BuildJavaVersion)
	}

	preferred := make([]javainfo.Installation, 0, 1)
	if detection.RuntimeJava.Home != "" {
		preferred = append(preferred, javainfo.Installation{
			Version: detection.RuntimeJava.Version,
			Home:    detection.RuntimeJava.Home,
		})
	}
	reportProgress(progress, ProgressEvent{
		Step: 4, Total: initializationSteps, Name: "JDK", State: ProgressStarted,
		Message: "Locate matching installed JDK",
	})
	java, err := i.java.Locate(ctx, detection.BuildJavaVersion, preferred...)
	if err != nil {
		reportFailure(progress, 4, "JDK", err)
		return Config{}, "", fmt.Errorf("locate project JDK: %w", err)
	}
	reportSuccess(progress, 4, "JDK", "Java "+java.Version+" at "+java.Home)

	config := Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   canonicalRoot,
		BuildTool:     detection.Tool,
		Java:          java,
		InitializedAt: i.now().UTC(),
	}
	reportProgress(progress, ProgressEvent{
		Step: 5, Total: initializationSteps, Name: "CONFIG", State: ProgressStarted,
		Message: "Save local project configuration",
	})
	path, err := i.store.Save(config)
	if err != nil {
		reportFailure(progress, 5, "CONFIG", err)
		return Config{}, "", err
	}
	reportSuccess(progress, 5, "CONFIG", path)
	return config, path, nil
}

func wrapperProgressMessage(info buildtool.Info) string {
	if !info.Wrapper.Enabled {
		return "Build wrapper: not detected"
	}
	return "Build wrapper: " + filepath.Base(info.Wrapper.Executable)
}

func reportSuccess(progress ProgressFunc, step int, name, message string) {
	reportProgress(progress, ProgressEvent{
		Step: step, Total: initializationSteps, Name: name, Message: message, State: ProgressSucceeded,
	})
}

func reportFailure(progress ProgressFunc, step int, name string, err error) {
	reportProgress(progress, ProgressEvent{
		Step: step, Total: initializationSteps, Name: name, Message: err.Error(), State: ProgressFailed,
	})
}

func canonicalProjectRoot(root string) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	if resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot); err == nil {
		absoluteRoot = resolvedRoot
	}
	return filepath.Clean(absoluteRoot), nil
}
