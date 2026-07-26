// Package javaversion parses the major component of Java version strings.
package javaversion

import (
	"fmt"
	"strings"
)

// Major returns the Java major version from modern and legacy version forms.
func Major(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "1.")
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return "", fmt.Errorf("unsupported Java version %q", value)
	}
	return value[:end], nil
}

// OptionalMajor accepts an empty version in addition to the forms Major accepts.
func OptionalMajor(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return Major(value)
}
