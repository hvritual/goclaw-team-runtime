package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestWorkspaceIssuePrefixMatchesRetainedCompatibility(t *testing.T) {
	tests := map[string]string{
		"New Team": "NEW",
		"1Team":    "TEA",
		"12":       "WS",
		"团队":       "WS",
	}
	for name, expected := range tests {
		if actual := workspaceIssuePrefix(name); actual != expected {
			t.Errorf("workspaceIssuePrefix(%q) = %q, want %q", name, actual, expected)
		}
	}
}

type recordingWorkspaceCreationRepository struct {
	called bool
}

func (r *recordingWorkspaceCreationRepository) Create(context.Context, WorkspaceCreation) (contract.WorkspaceSelection, error) {
	r.called = true
	return contract.WorkspaceSelection{}, nil
}

func TestWorkspaceCreationStopsBeforePersistenceWhenIDGenerationFails(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID func(context.Context) (string, error)
		memberID    func(context.Context) (string, error)
	}{
		{
			name:        "workspace id",
			workspaceID: func(context.Context) (string, error) { return "", errors.New("workspace id unavailable") },
			memberID:    func(context.Context) (string, error) { return "member-id", nil },
		},
		{
			name:        "member id",
			workspaceID: func(context.Context) (string, error) { return "workspace-id", nil },
			memberID:    func(context.Context) (string, error) { return "", errors.New("member id unavailable") },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &recordingWorkspaceCreationRepository{}
			useCase, err := NewWorkspaceCreationUseCase(repository, test.workspaceID, test.memberID, time.Now)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := useCase.Create(context.Background(), "user-id", contract.CreateWorkspaceRequest{Name: "Acme", Slug: "acme"}); err == nil {
				t.Fatal("Create() error = nil")
			}
			if repository.called {
				t.Fatal("repository called after ID generation failure")
			}
		})
	}
}
