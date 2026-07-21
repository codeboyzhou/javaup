package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/codeboyzhou/javaup/internal/selfupdate"
)

type stubUpdateService struct {
	checkResult  selfupdate.Result
	updateResult selfupdate.Result
	err          error
}

func (s stubUpdateService) Check(context.Context) (selfupdate.Result, error) {
	return s.checkResult, s.err
}

func (s stubUpdateService) Update(context.Context) (selfupdate.Result, error) {
	return s.updateResult, s.err
}

func TestUpdateCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		service stubUpdateService
		want    string
		wantErr string
	}{
		{
			name: "checks for an available update",
			args: []string{"--check"},
			service: stubUpdateService{checkResult: selfupdate.Result{
				Current: "v1.0.0", Latest: "v1.1.0", Updated: true,
			}},
			want: "Update available: v1.0.0 -> v1.1.0",
		},
		{
			name: "reports current version",
			service: stubUpdateService{updateResult: selfupdate.Result{
				Current: "v1.1.0", Latest: "v1.1.0",
			}},
			want: "Already up to date (v1.1.0)",
		},
		{
			name: "reports completed update",
			service: stubUpdateService{updateResult: selfupdate.Result{
				Current: "v1.0.0", Latest: "v1.1.0", Updated: true,
			}},
			want: "Updated jup from v1.0.0 to v1.1.0.",
		},
		{
			name:    "returns update errors",
			service: stubUpdateService{err: errors.New("network unavailable")},
			wantErr: "network unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			command := newUpdateCommand(func() updateService { return tt.service })
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
