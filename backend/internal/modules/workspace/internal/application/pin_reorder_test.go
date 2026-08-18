package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type recordingPinReorderRepository struct {
	ProjectSurfaceRepository
	workspaceID      string
	userID           string
	ids              []string
	expectedRevision int64
	err              error
	calls            int
}

func (r *recordingPinReorderRepository) ReorderPins(_ context.Context, workspaceID, userID string, ids []string, expectedRevision int64) error {
	r.calls++
	r.workspaceID, r.userID = workspaceID, userID
	r.ids = append([]string(nil), ids...)
	r.expectedRevision = expectedRevision
	return r.err
}

func TestProjectSurfaceReorderPinsValidatesAuthorizesAndDelegates(t *testing.T) {
	repository := &recordingPinReorderRepository{}
	authorizer := &recordingProjectSurfaceSearchAuthorizer{}
	service := &ProjectSurfaceUseCase{repository: repository, authorizer: authorizer}

	err := service.ReorderPins(context.Background(), " workspace-1 ", " user-1 ", contract.ReorderPinsRequest{
		Items: []contract.ReorderPinItem{{ID: " pin-2 "}, {ID: "pin-1"}}, ExpectedRevision: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.calls != 1 || repository.workspaceID != "workspace-1" || repository.userID != "user-1" || repository.expectedRevision != 7 {
		t.Fatalf("repository call = %+v", repository)
	}
	if len(repository.ids) != 2 || repository.ids[0] != "pin-2" || repository.ids[1] != "pin-1" {
		t.Fatalf("ids = %v", repository.ids)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != contract.PermissionPinReorder {
		t.Fatalf("permissions = %v", authorizer.permissions)
	}
}

func TestProjectSurfaceReorderPinsRejectsInvalidAndDeniedBeforeRepository(t *testing.T) {
	repository := &recordingPinReorderRepository{}
	authorizer := &recordingProjectSurfaceSearchAuthorizer{}
	service := &ProjectSurfaceUseCase{repository: repository, authorizer: authorizer}
	invalid := []contract.ReorderPinsRequest{
		{},
		{Items: []contract.ReorderPinItem{{ID: "pin-1"}}},
		{Items: []contract.ReorderPinItem{{ID: " "}}, ExpectedRevision: 1},
		{Items: []contract.ReorderPinItem{{ID: "pin-1"}, {ID: " pin-1 "}}, ExpectedRevision: 1},
	}
	for _, request := range invalid {
		if err := service.ReorderPins(context.Background(), "workspace-1", "user-1", request); !errors.Is(err, ErrInvalidProjectSurfaceRequest) {
			t.Fatalf("request %+v error = %v", request, err)
		}
	}
	if repository.calls != 0 || len(authorizer.permissions) != 0 {
		t.Fatalf("invalid request touched dependencies: calls=%d permissions=%v", repository.calls, authorizer.permissions)
	}
	authorizer.err = contract.ErrWorkspacePermissionDenied
	err := service.ReorderPins(context.Background(), "workspace-1", "user-1", contract.ReorderPinsRequest{
		Items: []contract.ReorderPinItem{{ID: "pin-1"}}, ExpectedRevision: 1,
	})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) || repository.calls != 0 {
		t.Fatalf("denied error/calls = %v/%d", err, repository.calls)
	}
}
