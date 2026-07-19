package project

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConfigUnmarshalsRFC3339InitializedAt(t *testing.T) {
	t.Parallel()

	var config Config
	if err := json.Unmarshal([]byte(`{"initializedAt":"2026-07-19T01:29:08+08:00"}`), &config); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := config.InitializedAt.Format(time.RFC3339), "2026-07-19T01:29:08+08:00"; got != want {
		t.Errorf("timestamp = %q, want %q", got, want)
	}
}
