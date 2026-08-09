package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspaceDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/workspace"
)

type workspaceIdentityRepositoryStub struct {
	workspace workspaceDomain.Workspace
	err       error
}

func (r workspaceIdentityRepositoryStub) FindByID(context.Context, string) (workspaceDomain.Workspace, error) {
	return r.workspace, r.err
}

func TestWorkspaceIdentityService(t *testing.T) {
	repositoryFailure := errors.New("repository unavailable")
	tests := []struct {
		name     string
		repo     workspaceIdentityRepositoryStub
		want     contract.WorkspaceIdentity
		wantErr  error
		exactErr error
	}{
		{
			name: "returns identity",
			repo: workspaceIdentityRepositoryStub{
				workspace: *workspaceDomain.New("workspace-1", "Acme"),
			},
			want: contract.WorkspaceIdentity{ID: "workspace-1", Name: "Acme"},
		},
		{
			name:    "maps missing workspace to contract error",
			repo:    workspaceIdentityRepositoryStub{err: ErrWorkspaceNotFound},
			wantErr: contract.ErrWorkspaceNotFound,
		},
		{
			name:     "propagates repository failure",
			repo:     workspaceIdentityRepositoryStub{err: repositoryFailure},
			exactErr: repositoryFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewWorkspaceIdentityService(tt.repo)
			got, err := service.FindIdentity(context.Background(), "workspace-1")
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("FindIdentity() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
			if tt.exactErr != nil && err != tt.exactErr {
				t.Fatalf("FindIdentity() error = %v, want exact %v", err, tt.exactErr)
			}
			if tt.wantErr == nil && tt.exactErr == nil && err != nil {
				t.Fatalf("FindIdentity() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("FindIdentity() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
