package project

import (
	"context"
	"testing"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
)

type fakeDetector struct {
	detection buildtool.Detection
}

func (d fakeDetector) Detect(context.Context, string) (buildtool.Detection, bool, error) {
	return d.detection, true, nil
}

type fakeJavaLocator struct {
	requested string
	result    javainfo.Installation
}

func (l *fakeJavaLocator) Locate(_ context.Context, version string, _ ...javainfo.Installation) (javainfo.Installation, error) {
	l.requested = version
	return l.result, nil
}

type fakeConfigStore struct {
	config Config
	path   string
}

func (s *fakeConfigStore) Save(config Config) (string, error) {
	s.config = config
	return s.path, nil
}

func TestInitializerCoordinatesDetectionAndStorage(t *testing.T) {
	t.Parallel()

	detection := buildtool.Detection{
		Tool: buildtool.Info{
			Type:    buildtool.Maven,
			Version: "3.9.11",
			Wrapper: buildtool.Wrapper{Enabled: true, Executable: "mvnw"},
		},
		BuildJavaVersion: "17",
	}
	java := &fakeJavaLocator{result: javainfo.Installation{Version: "17", Home: "/jdk-17"}}
	store := &fakeConfigStore{path: "/config/project.json"}
	initializer := NewInitializer([]buildtool.Detector{fakeDetector{detection: detection}}, java, store)
	initializedAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.FixedZone("test", 8*60*60))
	initializer.now = func() time.Time { return initializedAt }
	var events []ProgressEvent

	config, path, err := initializer.Initialize(context.Background(), t.TempDir(), func(event ProgressEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if java.requested != "17" {
		t.Errorf("requested Java = %q, want %q", java.requested, "17")
	}
	if config.Java.Home != "/jdk-17" {
		t.Errorf("Java home = %q, want %q", config.Java.Home, "/jdk-17")
	}
	if config.InitializedAt != initializedAt.UTC() {
		t.Errorf("InitializedAt = %v, want %v", config.InitializedAt, initializedAt.UTC())
	}
	if path != store.path {
		t.Errorf("path = %q, want %q", path, store.path)
	}
	if store.config.SchemaVersion != currentSchemaVersion {
		t.Errorf("stored schema version = %d, want %d", store.config.SchemaVersion, currentSchemaVersion)
	}

	wantEvents := []struct {
		name  string
		state ProgressState
	}{
		{name: projectStepName, state: ProgressStarted},
		{name: projectStepName, state: ProgressSucceeded},
		{name: buildToolStepName, state: ProgressStarted},
		{name: buildToolStepName, state: ProgressSucceeded},
		{name: javaVersionStepName, state: ProgressStarted},
		{name: javaVersionStepName, state: ProgressSucceeded},
		{name: jdkStepName, state: ProgressStarted},
		{name: jdkStepName, state: ProgressSucceeded},
		{name: configStepName, state: ProgressStarted},
		{name: configStepName, state: ProgressSucceeded},
	}
	if len(events) != len(wantEvents) {
		t.Fatalf("progress event count = %d, want %d", len(events), len(wantEvents))
	}
	for index, want := range wantEvents {
		if events[index].Name != want.name || events[index].State != want.state {
			t.Errorf(
				"progress event %d = %s/%s, want %s/%s",
				index, events[index].Name, events[index].State, want.name, want.state,
			)
		}
		if events[index].Step < 1 || events[index].Total != initializationSteps {
			t.Errorf("progress event %d has invalid position %d/%d", index, events[index].Step, events[index].Total)
		}
	}
	wantStartedMessages := map[string]string{
		projectStepName:     "Inspecting current project directory",
		buildToolStepName:   "Detecting build tool, version, and wrapper",
		javaVersionStepName: "Detecting configured Java build version",
		jdkStepName:         "Locating matching installed JDK",
		configStepName:      "Saving local project configuration",
	}
	for _, event := range events {
		if event.State == ProgressStarted && event.Message != wantStartedMessages[event.Name] {
			t.Errorf("%s progress message = %q, want %q", event.Name, event.Message, wantStartedMessages[event.Name])
		}
	}
	if events[3].Message != "Maven 3.9.11 (wrapper)" {
		t.Errorf("build tool progress = %q, want %q", events[3].Message, "Maven 3.9.11 (wrapper)")
	}
}

func TestBuildToolProgressMessageIdentifiesSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wrapper bool
		want    string
	}{
		{name: "wrapper", wrapper: true, want: "Maven 3.9.16 (wrapper)"},
		{name: "path", wrapper: false, want: "Maven 3.9.16 (PATH)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			info := buildtool.Info{
				Type: buildtool.Maven, Version: "3.9.16", Wrapper: buildtool.Wrapper{Enabled: test.wrapper},
			}
			if got := buildToolProgressMessage(info); got != test.want {
				t.Errorf("buildToolProgressMessage() = %q, want %q", got, test.want)
			}
		})
	}
}
