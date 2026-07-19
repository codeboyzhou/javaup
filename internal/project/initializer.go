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

type initializerConfigStore interface {
	configStore
	Load(projectRoot string) (config Config, path string, found bool, err error)
}

// Initializer detects a project's build environment and persists it locally.
type Initializer struct {
	detectors []buildtool.Detector
	java      javaLocator
	store     initializerConfigStore
	now       func() time.Time
}

const (
	initializationSteps = 5
	buildToolStepName   = "Build Tool"
	javaVersionStepName = "Java Version"
	jdkStepName         = "JDK"
)

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
func NewInitializer(detectors []buildtool.Detector, java javaLocator, store initializerConfigStore) *Initializer {
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
		Step: 1, Total: initializationSteps, Name: projectStepName, State: ProgressStarted,
		Message: "Inspecting current project directory",
	})
	canonicalRoot, err := canonicalProjectRoot(root)
	if err != nil {
		reportFailure(progress, 1, projectStepName, err)
		return Config{}, "", err
	}
	reportSuccess(progress, 1, projectStepName, canonicalRoot)

	reportProgress(progress, ProgressEvent{
		Step: 2, Total: initializationSteps, Name: buildToolStepName, State: ProgressStarted,
		Message: "Detecting build tool, version, and wrapper",
	})
	var detection buildtool.Detection
	found := false
	for _, detector := range i.detectors {
		detection, found, err = detector.Detect(ctx, canonicalRoot)
		if err != nil {
			reportFailure(progress, 2, buildToolStepName, err)
			return Config{}, "", err
		}
		if found {
			break
		}
	}
	if !found {
		err := fmt.Errorf("no supported build project found in %s", canonicalRoot)
		reportFailure(progress, 2, buildToolStepName, err)
		return Config{}, "", err
	}
	reportSuccess(progress, 2, buildToolStepName, buildToolProgressMessage(detection.Tool))

	reportProgress(progress, ProgressEvent{
		Step: 3, Total: initializationSteps, Name: javaVersionStepName, State: ProgressStarted,
		Message: "Detecting configured Java build version",
	})
	if detection.BuildJavaVersion == "" {
		reportSuccess(progress, 3, javaVersionStepName, "Use the build runtime JDK")
	} else {
		reportSuccess(progress, 3, javaVersionStepName, "Java "+detection.BuildJavaVersion)
	}

	preferred := make([]javainfo.Installation, 0, 1)
	if detection.RuntimeJava.Home != "" {
		preferred = append(preferred, javainfo.Installation{
			Version: detection.RuntimeJava.Version,
			Home:    detection.RuntimeJava.Home,
		})
	}
	reportProgress(progress, ProgressEvent{
		Step: 4, Total: initializationSteps, Name: jdkStepName, State: ProgressStarted,
		Message: "Locating matching installed JDK",
	})
	java, err := i.java.Locate(ctx, detection.BuildJavaVersion, preferred...)
	if err != nil {
		reportFailure(progress, 4, jdkStepName, err)
		return Config{}, "", fmt.Errorf("locate project JDK: %w", err)
	}
	reportSuccess(progress, 4, jdkStepName, "Java "+java.Version+" at "+java.Home)

	config := Config{
		SchemaVersion: currentSchemaVersion,
		ProjectRoot:   canonicalRoot,
		BuildTool:     detection.Tool,
		Java:          java,
		InitializedAt: i.now().Truncate(time.Second),
	}
	i.preserveMavenSettingsBinding(&config)
	reportProgress(progress, ProgressEvent{
		Step: 5, Total: initializationSteps, Name: configStepName, State: ProgressStarted,
		Message: "Saving local project configuration",
	})
	path, err := i.store.Save(config)
	if err != nil {
		reportFailure(progress, 5, configStepName, err)
		return Config{}, "", err
	}
	reportSuccess(progress, 5, configStepName, path)
	return config, path, nil
}

func (i *Initializer) preserveMavenSettingsBinding(config *Config) {
	if config.BuildTool.Type != buildtool.Maven {
		return
	}

	existing, _, found, err := i.store.Load(config.ProjectRoot)
	if err != nil || !found || existing.BuildTool.Type != buildtool.Maven {
		return
	}
	config.BuildTool.SettingsAlias = existing.BuildTool.SettingsAlias
}

func buildToolProgressMessage(info buildtool.Info) string {
	return info.Summary()
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
