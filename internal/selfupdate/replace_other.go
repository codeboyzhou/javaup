//go:build !windows

package selfupdate

import (
	"fmt"
	"os"
)

func applyUpdate(staged, target string) (bool, error) {
	if err := os.Rename(staged, target); err != nil {
		return false, fmt.Errorf("atomically install update: %w", err)
	}
	return false, nil
}
