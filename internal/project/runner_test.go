package project

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
	"github.com/codeboyzhou/javaup/internal/mavensettings"
)

type fakeConfigFinder struct {
	config Config
	found  bool
	err    error
	start  string
}

func (f *fakeConfigFinder) Find(start string) (Config, string, bool, error) {
	f.start = start
	return f.config, "config.json", f.found, f.err
}

type recordingProcessExecutor struct {
	spec processSpec
	err  error
}

type recordingUsageStore struct {
	root string
	at   time.Time
	err  error
}

func (s *recordingUsageStore) Touch(_ context.Context, root string, at time.Time) error {
	s.root = root
	s.at = at
	return s.err
}

func (e *recordingProcessExecutor) Execute(_ context.Context, spec processSpec) error {
	e.spec = spec
	return e.err
}

func TestRunnerRunsConfiguredMavenWithProjectJava(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	javaHome := filepath.Join(root, "jdk-17")
	mavenPath := filepath.Join(root, "maven", mavenExecutableName())
	writeRunnerExecutable(t, mavenPath)
	finder := &fakeConfigFinder{
		found: true,
		config: Config{
			BuildTool: buildtool.Info{
				Type:       buildtool.Maven,
				Version:    "3.9.11",
				Executable: mavenPath,
			},
			Java: javainfo.Installation{Version: "17", Home: javaHome},
		},
	}
	executor := &recordingProcessExecutor{}
	runner := NewRunner(finder, executor)

	streams := Streams{Stdin: strings.NewReader("input"), Stdout: io.Discard, Stderr: io.Discard}
	if err := runner.Run(context.Background(), root, buildtool.Maven, []string{"test", "-DskipTests"}, streams); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if finder.start != root {
		t.Errorf("Find() start = %q, want %q", finder.start, root)
	}
	if executor.spec.executable != mavenPath {
		t.Errorf("executable = %q, want %q", executor.spec.executable, mavenPath)
	}
	if got := strings.Join(executor.spec.args, " "); got != "test -DskipTests" {
		t.Errorf("args = %q, want %q", got, "test -DskipTests")
	}
	assertEnvironmentValue(t, executor.spec.environment, "JAVA_HOME", javaHome)
	pathValue := environmentValue(executor.spec.environment, "PATH")
	if first := strings.Split(pathValue, string(os.PathListSeparator))[0]; !samePath(first, filepath.Join(javaHome, "bin")) {
		t.Errorf("PATH first entry = %q, want %q", first, filepath.Join(javaHome, "bin"))
	}
	if executor.spec.streams != streams {
		t.Error("process streams were not forwarded")
	}
}

func TestRunnerRecordsSelectedProjectBeforeExecution(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mavenPath := filepath.Join(root, "maven", mavenExecutableName())
	writeRunnerExecutable(t, mavenPath)
	usage := &recordingUsageStore{}
	at := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	runner := NewRunner(&fakeConfigFinder{
		found: true,
		config: Config{
			ProjectRoot: root,
			BuildTool:   buildtool.Info{Type: buildtool.Maven, Version: "3.9.11", Executable: mavenPath},
			Java:        javainfo.Installation{Version: "21", Home: filepath.Join(root, "jdk")},
		},
	}, &recordingProcessExecutor{})
	runner.usage = usage
	runner.now = func() time.Time { return at }

	if err := runner.Run(context.Background(), root, buildtool.Maven, []string{"test"}, Streams{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if usage.root != root || !usage.at.Equal(at) {
		t.Errorf("Touch() root/time = %q/%v, want %q/%v", usage.root, usage.at, root, at)
	}
}

func TestRunnerUsesBuildToolSettingsAlias(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mavenPath := filepath.Join(root, "maven", mavenExecutableName())
	writeRunnerExecutable(t, mavenPath)
	resolver := &fakeMavenSettingsResolver{
		entry: mavensettings.Entry{Alias: "intranet", Path: filepath.Join(root, "settings.xml")},
	}
	executor := &recordingProcessExecutor{}
	runner := NewRunnerWithMavenSettings(&fakeConfigFinder{
		found: true,
		config: Config{
			BuildTool: buildtool.Info{
				Type:          buildtool.Maven,
				Version:       "3.9.11",
				Executable:    mavenPath,
				SettingsAlias: "intranet",
			},
			Java: javainfo.Installation{
				Version: "17",
				Home:    filepath.Join(root, "jdk-17"),
			},
		},
	}, resolver, executor)

	if err := runner.Run(context.Background(), root, buildtool.Maven, []string{"clean", "package"}, Streams{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resolver.alias != "intranet" {
		t.Errorf("Resolve() alias = %q, want intranet", resolver.alias)
	}
	want := []string{"--settings", resolver.entry.Path, "clean", "package"}
	if !reflect.DeepEqual(executor.spec.args, want) {
		t.Errorf("process args = %#v, want %#v", executor.spec.args, want)
	}
}

func TestRunnerExecutionIsGenericAcrossBuildToolTypes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	const gradle buildtool.Type = "gradle"
	executable := filepath.Join(root, "gradle", "bin", "gradle")
	writeRunnerExecutable(t, executable)
	executor := &recordingProcessExecutor{}
	runner := NewRunner(&fakeConfigFinder{
		found: true,
		config: Config{
			BuildTool: buildtool.Info{Type: gradle, Version: "9.0", Executable: executable},
			Java:      javainfo.Installation{Version: "21", Home: filepath.Join(root, "jdk")},
		},
	}, executor)

	if err := runner.Run(context.Background(), root, gradle, []string{"build"}, Streams{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executor.spec.executable != executable || strings.Join(executor.spec.args, " ") != "build" {
		t.Errorf("process executable/args = %q/%#v", executor.spec.executable, executor.spec.args)
	}
}

func TestRunnerRequiresInitializedProject(t *testing.T) {
	t.Parallel()

	runner := NewRunner(&fakeConfigFinder{}, &recordingProcessExecutor{})
	err := runner.Run(context.Background(), t.TempDir(), buildtool.Maven, nil, Streams{})
	if err == nil || !strings.Contains(err.Error(), "run jup init") {
		t.Fatalf("Run() error = %v, want initialization guidance", err)
	}
}

func TestRunnerRejectsMissingConfiguredExecutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runner := NewRunner(&fakeConfigFinder{
		found: true,
		config: Config{
			BuildTool: buildtool.Info{
				Type:       buildtool.Maven,
				Version:    "3.9.11",
				Executable: filepath.Join(root, "missing-maven", mavenExecutableName()),
			},
			Java: javainfo.Installation{Version: "21", Home: filepath.Join(root, "jdk")},
		},
	}, &recordingProcessExecutor{})
	err := runner.Run(context.Background(), root, buildtool.Maven, nil, Streams{})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("Run() error = %v, want missing executable error", err)
	}
}

func assertEnvironmentValue(t *testing.T, environment []string, name, want string) {
	t.Helper()
	if got := environmentValue(environment, name); got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

func environmentValue(environment []string, target string) string {
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if found && environmentNameEqual(name, target) {
			return value
		}
	}
	return ""
}

func writeRunnerExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func mavenExecutableName() string {
	if runtime.GOOS == "windows" {
		return "mvn.cmd"
	}
	return "mvn"
}
