package maven

import (
	"path/filepath"
	"testing"
)

func TestDetectBuildJavaVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pom  string
		want string
	}{
		{
			name: "compiler release property",
			pom:  `<project><properties><maven.compiler.release>17</maven.compiler.release></properties></project>`,
			want: "17",
		},
		{
			name: "java version property",
			pom:  `<project><properties><java.version>21.0.2</java.version></properties></project>`,
			want: "21",
		},
		{
			name: "legacy compiler target",
			pom:  `<project><properties><maven.compiler.source>1.8</maven.compiler.source><maven.compiler.target>1.8</maven.compiler.target></properties></project>`,
			want: "8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pomPath := filepath.Join(t.TempDir(), "pom.xml")
			writeTestFile(t, pomPath, tt.pom)
			got, err := detectBuildJavaVersion(pomPath)
			if err != nil {
				t.Fatalf("detectBuildJavaVersion() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("detectBuildJavaVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectBuildJavaVersionInheritsParentConfiguration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pom.xml"), `<project><properties><java.version>11</java.version></properties><build><pluginManagement><plugins><plugin><artifactId>maven-compiler-plugin</artifactId><configuration><release>${java.version}</release></configuration></plugin></plugins></pluginManagement></build></project>`)
	childPOM := filepath.Join(root, "module", "pom.xml")
	writeTestFile(t, childPOM, `<project><parent><groupId>example</groupId><artifactId>parent</artifactId><version>1</version></parent><properties><java.version>21</java.version></properties></project>`)

	got, err := detectBuildJavaVersion(childPOM)
	if err != nil {
		t.Fatalf("detectBuildJavaVersion() error = %v", err)
	}
	if got != "21" {
		t.Errorf("detectBuildJavaVersion() = %q, want %q", got, "21")
	}
}
