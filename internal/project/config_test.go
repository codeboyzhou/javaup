package project

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLocalTimestampUnmarshalsLegacyRFC3339(t *testing.T) {
	t.Parallel()

	var timestamp LocalTimestamp
	if err := json.Unmarshal([]byte(`"2026-07-19T00:04:05+08:00"`), &timestamp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := timestamp.Format(time.RFC3339), "2026-07-19T00:04:05+08:00"; got != want {
		t.Errorf("timestamp = %q, want %q", got, want)
	}
}
