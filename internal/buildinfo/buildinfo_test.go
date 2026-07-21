package buildinfo

import "testing"

func TestNewInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		commit     string
		wantCommit string
	}{
		{
			name:       "shortens a full revision",
			commit:     "64c2fb07bcad179743ddf04e31d0a610c28d1344",
			wantCommit: "64c2fb07bcad",
		},
		{
			name:       "preserves a short revision",
			commit:     "abcdef0",
			wantCommit: "abcdef0",
		},
		{
			name:       "uses unknown when revision is unavailable",
			wantCommit: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newInfo("v0.2.0", "testos", "testarch", tt.commit)
			if got.Version != "v0.2.0" {
				t.Errorf("Version = %q, want %q", got.Version, "v0.2.0")
			}
			if got.Platform != "testos/testarch" {
				t.Errorf("Platform = %q, want %q", got.Platform, "testos/testarch")
			}
			if got.Commit != tt.wantCommit {
				t.Errorf("Commit = %q, want %q", got.Commit, tt.wantCommit)
			}
		})
	}
}
