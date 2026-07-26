// Package atomicfile writes files through a temporary sibling and rename.
package atomicfile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Write replaces path only after write has completed and the temporary file
// has been synchronized. A non-zero mode is applied before writing.
func Write(path, pattern string, mode os.FileMode, write func(io.Writer) error) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	replaced := false
	defer func() {
		_ = temporary.Close()
		if !replaced {
			_ = os.Remove(temporaryPath)
		}
	}()

	if mode != 0 {
		if err := temporary.Chmod(mode); err != nil {
			return fmt.Errorf("set temporary file permissions: %w", err)
		}
	}
	if err := write(temporary); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace destination file: %w", err)
	}
	replaced = true
	return nil
}

// WriteJSON atomically writes indented, human-readable JSON.
func WriteJSON(path, pattern string, value any) error {
	return Write(path, pattern, 0, func(writer io.Writer) error {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(value)
	})
}

// WriteBytes atomically writes contents while applying mode when non-zero.
func WriteBytes(path, pattern string, contents []byte, mode os.FileMode) error {
	return Write(path, pattern, mode, func(writer io.Writer) error {
		_, err := writer.Write(contents)
		return err
	})
}
