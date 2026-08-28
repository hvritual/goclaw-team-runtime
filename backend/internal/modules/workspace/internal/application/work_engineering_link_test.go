package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type engineeringWorkReaderStub struct {
	exists bool
	calls  int
}

func (s *engineeringWorkReaderStub) WorkExists(context.Context, string, contract.EngineeringWorkKind, string) (bool, error) {
	s.calls++
	return s.exists, nil
}

type membershipReaderStub struct{}

func (membershipReaderStub) ListForUser(context.Context, string) ([]contract.WorkspaceMembership, error) {
	return nil, nil
}
func (membershipReaderStub) FindForUserAndWorkspace(_ context.Context, userID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	role := map[string]string{"owner-user": "owner", "admin-user": "admin", "member-user": "member"}[userID]
	if role == "" || workspaceID != "workspace-a" {
		return contract.WorkspaceMembership{}, false, nil
	}
	return contract.WorkspaceMembership{MemberID: "member-" + userID, UserID: userID, WorkspaceID: workspaceID, Role: role}, true, nil
}
func (membershipReaderStub) FindByMemberAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return contract.WorkspaceMembership{}, false, nil
}

type engineeringGatewayStub struct {
	putCalls    int
	listCalls   int
	deleteCalls int
	lastKind    contract.EngineeringWorkKind
}

func (s *engineeringGatewayStub) PutEngineeringWorkLink(_ context.Context, workspaceID string, kind contract.EngineeringWorkKind, workID, entityID string) (contract.EngineeringWorkLink, error) {
	s.putCalls++
	s.lastKind = kind
	return contract.EngineeringWorkLink{ID: "link-1", WorkspaceID: workspaceID, WorkKind: kind, WorkID: workID, EntityID: entityID}, nil
}
func (s *engineeringGatewayStub) ListEngineeringWorkLinks(_ context.Context, workspaceID string, kind contract.EngineeringWorkKind, workID string) ([]contract.EngineeringWorkLink, error) {
	s.listCalls++
	return []contract.EngineeringWorkLink{{ID: "link-1", WorkspaceID: workspaceID, WorkKind: kind, WorkID: workID, EntityID: "service-1"}}, nil
}
func (s *engineeringGatewayStub) DeleteEngineeringWorkLink(context.Context, string, contract.EngineeringWorkKind, string, string) error {
	s.deleteCalls++
	return nil
}

func TestWorkEngineeringLinkUseCaseAuthorizationAndReferenceValidation(t *testing.T) {
	work := &engineeringWorkReaderStub{exists: true}
	gateway := &engineeringGatewayStub{}
	usecase, err := NewWorkEngineeringLinkUseCase(work, membershipReaderStub{}, gateway)
	if err != nil {
		t.Fatal(err)
	}
	owner := contract.WithWorkspaceActor(context.Background(), "member", "owner-user")
	link, err := usecase.LinkEngineeringEntity(owner, "workspace-a", contract.EngineeringWorkProject, "project-1", "service-1")
	if err != nil {
		t.Fatal(err)
	}
	if link.WorkKind != contract.EngineeringWorkProject || gateway.putCalls != 1 || work.calls != 1 {
		t.Fatalf("link=%+v putCalls=%d workCalls=%d", link, gateway.putCalls, work.calls)
	}

	member := contract.WithWorkspaceActor(context.Background(), "member", "member-user")
	if _, err := usecase.LinkEngineeringEntity(member, "workspace-a", contract.EngineeringWorkTask, "task-1", "service-1"); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("member write error=%v", err)
	}
	if _, err := usecase.ListEngineeringLinks(member, "workspace-a", contract.EngineeringWorkProject, "project-1"); err != nil {
		t.Fatalf("member read error=%v", err)
	}

	work.exists = false
	if _, err := usecase.LinkEngineeringEntity(owner, "workspace-a", contract.EngineeringWorkRequirement, "missing", "service-1"); !errors.Is(err, contract.ErrEngineeringWorkNotFound) {
		t.Fatalf("missing work error=%v", err)
	}
	if gateway.putCalls != 1 {
		t.Fatalf("gateway put calls=%d after missing work", gateway.putCalls)
	}
}

func TestWorkEngineeringLinkUseCaseRetainsLinksAfterSourceRemovalUntilExplicitUnlink(t *testing.T) {
	work := &engineeringWorkReaderStub{exists: false}
	gateway := &engineeringGatewayStub{}
	usecase, err := NewWorkEngineeringLinkUseCase(work, membershipReaderStub{}, gateway)
	if err != nil {
		t.Fatal(err)
	}
	owner := contract.WithWorkspaceActor(context.Background(), "member", "owner-user")
	links, err := usecase.ListEngineeringLinks(owner, "workspace-a", contract.EngineeringWorkTask, "deleted-task")
	if err != nil || len(links) != 1 {
		t.Fatalf("historical links=%+v err=%v", links, err)
	}
	if work.calls != 0 || gateway.listCalls != 1 {
		t.Fatalf("historical list workCalls=%d gatewayCalls=%d", work.calls, gateway.listCalls)
	}
	if err := usecase.UnlinkEngineeringEntity(owner, "workspace-a", contract.EngineeringWorkTask, "deleted-task", "service-1"); err != nil {
		t.Fatal(err)
	}
	if work.calls != 0 || gateway.deleteCalls != 1 {
		t.Fatalf("unlink workCalls=%d deleteCalls=%d", work.calls, gateway.deleteCalls)
	}
}
