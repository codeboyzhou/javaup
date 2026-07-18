// Package project coordinates project detection and local configuration.
package project

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
)

const (
	currentSchemaVersion = 1
	localTimestampLayout = "2006-01-02 15:04:05"
)

// LocalTimestamp persists a time as a human-readable local date and time.
type LocalTimestamp struct {
	time.Time
}

// NewLocalTimestamp creates a timestamp that retains the supplied local clock time.
func NewLocalTimestamp(value time.Time) LocalTimestamp {
	return LocalTimestamp{Time: value}
}

// MarshalJSON writes the local clock time without a timezone suffix.
func (t LocalTimestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(localTimestampLayout))
}

// UnmarshalJSON accepts the local format and the legacy RFC 3339 format.
func (t *LocalTimestamp) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode local timestamp: %w", err)
	}
	parsed, err := time.ParseInLocation(localTimestampLayout, value, time.Local)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
	}
	if err != nil {
		return fmt.Errorf("parse local timestamp %q: %w", value, err)
	}
	t.Time = parsed
	return nil
}

// Config is the persisted description of an initialized project.
type Config struct {
	SchemaVersion int                   `json:"schemaVersion"`
	ProjectRoot   string                `json:"projectRoot"`
	BuildTool     buildtool.Info        `json:"buildTool"`
	Java          javainfo.Installation `json:"java"`
	InitializedAt LocalTimestamp        `json:"initializedAt"`
}
