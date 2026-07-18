package maven

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fakeRunner struct {
	output []byte
	calls  int
}

func (r *fakeRunner) Run(_ context.Context, _, _ string) ([]byte, error) {
	r.calls++
	return r.output, nil
}

func TestDetectorDetectsMavenWrapper(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pom.xml"), `<project><properties><maven.compiler.release>17</maven.compiler.release></properties></project>`)
	writeTestFile(t, filepath.Join(root, "mvnw"), "#!/bin/sh\n")
	writeTestFile(t, filepath.Join(root, "mvnw.cmd"), "@echo off\r\n")
	writeTestFile(t, filepath.Join(root, ".mvn", "wrapper", "maven-wrapper.properties"), `distributionUrl=https\://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/3.9.11/apache-maven-3.9.11-bin.zip`)

	runner := &fakeRunner{}
	detector := &Detector{runner: runner}
	detection, found, err := detector.Detect(context.Background(), root)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !found {
		t.Fatal("Detect() found = false, want true")
	}
	if detection.Tool.Version != "3.9.11" {
		t.Errorf("Maven version = %q, want %q", detection.Tool.Version, "3.9.11")
	}
	if !detection.Tool.Wrapper.Enabled {
		t.Error("Wrapper.Enabled = false, want true")
	}
	if detection.BuildJavaVersion != "17" {
		t.Errorf("BuildJavaVersion = %q, want %q", detection.BuildJavaVersion, "17")
	}
	if runner.calls != 0 {
		t.Errorf("wrapper runner calls = %d, want 0", runner.calls)
	}
}

func TestDetectorUsesInstalledMaven(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pom.xml"), `<project><build><plugins><plugin><artifactId>maven-compiler-plugin</artifactId><configuration><release>21</release></configuration></plugin></plugins></build></project>`)
	runner := &fakeRunner{output: []byte("Apache Maven 3.9.9 (test)\nMaven home: /opt/maven\nJava version: 21.0.7, vendor: Test, runtime: /opt/jdk-21\n")}
	detector := &Detector{runner: runner}

	detection, found, err := detector.Detect(context.Background(), root)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !found {
		t.Fatal("Detect() found = false, want true")
	}
	if detection.Tool.Version != "3.9.9" {
		t.Errorf("Maven version = %q, want %q", detection.Tool.Version, "3.9.9")
	}
	if detection.Tool.Wrapper.Enabled {
		t.Error("Wrapper.Enabled = true, want false")
	}
	if detection.BuildJavaVersion != "21" {
		t.Errorf("BuildJavaVersion = %q, want %q", detection.BuildJavaVersion, "21")
	}
	if detection.RuntimeJava.Version != "21" || detection.RuntimeJava.Home != filepath.Clean("/opt/jdk-21") {
		t.Errorf("RuntimeJava = %#v", detection.RuntimeJava)
	}
}

func TestDetectorIgnoresNonMavenProject(t *testing.T) {
	t.Parallel()

	detector := &Detector{runner: &fakeRunner{}}
	_, found, err := detector.Detect(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if found {
		t.Fatal("Detect() found = true, want false")
	}
}

func TestMavenVersionCommandRunsWindowsBatchFile(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows batch compatibility test")
	}

	executable := filepath.Join(t.TempDir(), "maven test", "mvn.cmd")
	writeTestFile(t, executable, "@echo off\r\necho Apache Maven 3.9.9 (test)\r\necho Java version: 17.0.12, vendor: Test, runtime: C:\\Java\\jdk-17\r\n")
	output, err := platformMavenVersionCommand(context.Background(), executable).CombinedOutput()
	if err != nil {
		t.Fatalf("mavenVersionCommand() error = %v, output = %s", err, output)
	}
	if !strings.Contains(string(output), "Apache Maven 3.9.9") {
		t.Errorf("output = %q", output)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
