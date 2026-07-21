package uninstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const installerProfileHeader = "# Added by the javaup installer"

func cleanInstallerProfile(path string, recognizedLines, removeLines map[string]bool) error {
	contents, err := os.ReadFile(path) // #nosec G304 -- paths are fixed beneath the resolved user home.
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read shell profile %s: %w", path, err)
	}
	newline := "\n"
	if bytes.Contains(contents, []byte("\r\n")) {
		newline = "\r\n"
	}
	normalized := strings.ReplaceAll(string(contents), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	cleaned := make([]string, 0, len(lines))
	changed := false
	for index := 0; index < len(lines); {
		if lines[index] != installerProfileHeader {
			cleaned = append(cleaned, lines[index])
			index++
			continue
		}
		next := index + 1
		for next < len(lines) && recognizedLines[lines[next]] {
			next++
		}
		if next == index+1 {
			cleaned = append(cleaned, lines[index])
			index++
			continue
		}
		remaining := make([]string, 0, next-index-1)
		for _, line := range lines[index+1 : next] {
			if removeLines[line] {
				changed = true
			} else {
				remaining = append(remaining, line)
			}
		}
		if len(remaining) > 0 {
			cleaned = append(cleaned, installerProfileHeader)
			cleaned = append(cleaned, remaining...)
		} else if len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
			cleaned = cleaned[:len(cleaned)-1]
		}
		index = next
	}
	if !changed {
		return nil
	}
	return replaceFile(path, []byte(strings.Join(cleaned, newline)))
}

func replaceFile(path string, contents []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".javaup-profile-*")
	if err != nil {
		return fmt.Errorf("create temporary shell profile: %w", err)
	}
	temporaryPath := temporary.Name()
	clean := true
	defer func() {
		_ = temporary.Close()
		if clean {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("preserve shell profile permissions: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		return fmt.Errorf("write shell profile: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close shell profile: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace shell profile: %w", err)
	}
	clean = false
	return nil
}
