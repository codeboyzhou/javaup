package atomicfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReplacesFileAfterSuccessfulWrite(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "value.txt")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteBytes(path, ".value-*", []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteBytes() error = %v", err)
	}
	// #nosec G304 -- path is test-controlled inside t.TempDir().
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "new" {
		t.Errorf("contents = %q, want new", contents)
	}
}

func TestWritePreservesDestinationAfterWriteFailure(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "value.txt")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("failed")
	err := Write(path, ".value-*", 0, func(io.Writer) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("Write() error = %v, want %v", err, want)
	}
	// #nosec G304 -- path is test-controlled inside t.TempDir().
	contents, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(contents) != "old" {
		t.Errorf("contents = %q, want old", contents)
	}
	entries, readDirErr := os.ReadDir(directory)
	if readDirErr != nil {
		t.Fatal(readDirErr)
	}
	if len(entries) != 1 {
		t.Errorf("directory entries = %d, want only destination", len(entries))
	}
}
