package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/uninstall"
)

type stubApplicationUninstaller struct {
	result uninstall.Result
	err    error
}

func (s stubApplicationUninstaller) Run(context.Context) (uninstall.Result, error) {
	return s.result, s.err
}

func TestUninstallCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		result     uninstall.Result
		serviceErr error
		wantPurge  bool
		want       string
		wantErr    string
	}{
		{
			name:   "preserves configuration by default",
			result: uninstall.Result{Home: "/home/test/.javaup"},
			want:   "Uninstalled jup. Configuration remains at /home/test/.javaup.",
		},
		{
			name:      "purges all data",
			args:      []string{"--purge"},
			result:    uninstall.Result{Home: "/home/test/.javaup", Purged: true},
			wantPurge: true,
			want:      "Uninstalled jup and removed all javaup data.",
		},
		{
			name:   "reports a scheduled windows uninstall",
			result: uninstall.Result{Home: `C:\Users\test\.javaup`, Pending: true},
			want:   "Uninstall scheduled",
		},
		{
			name:       "returns errors",
			serviceErr: errors.New("not managed"),
			wantErr:    "not managed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			command := newUninstallCommand(func(purge bool) applicationUninstaller {
				if purge != tt.wantPurge {
					t.Errorf("purge = %t, want %t", purge, tt.wantPurge)
				}
				return stubApplicationUninstaller{result: tt.result, err: tt.serviceErr}
			})
			command.SetOut(&output)
			command.SetArgs(tt.args)
			err := command.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(output.String(), tt.want) {
				t.Errorf("output = %q, want %q", output.String(), tt.want)
			}
		})
	}
}
