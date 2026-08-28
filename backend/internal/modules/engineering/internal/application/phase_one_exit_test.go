package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

func TestPhaseOneExitAcceptsChangeAndFreezesImmutableContextPack(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	member := contract.Actor{UserID: "member"}

	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{
		ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateChange(ctx, owner, "workspace-a", contract.CreateChangeRequest{
		ID:                "change-1",
		ProjectID:         "project-1",
		RequirementID:     "requirement-1",
		WorkItem:          &contract.NodeRef{Kind: "task", ID: "task-1"},
		Summary:           "Update reconnect handling",
		AffectedEntityIDs: []string{"service-1"},
		Artifacts:         []contract.ArtifactRef{{Kind: "pull_request", Locator: "github://acme/device-gateway/pull/7", Revision: "abc123"}},
		Provenance:        contract.Provenance{SourceType: "workspace", Locator: "workspace://workspace-a/tasks/task-1", Revision: "1"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := service.AcceptChange(ctx, member, "workspace-a", "change-1"); !errors.Is(err, contract.ErrForbidden) {
		t.Fatalf("member accept error = %v, want forbidden", err)
	}
	accepted, err := service.AcceptChange(ctx, owner, "workspace-a", "change-1")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "accepted" || accepted.AcceptedAt == nil {
		t.Fatalf("accepted change = %+v", accepted)
	}
	if _, err := service.AcceptChange(ctx, owner, "workspace-a", "change-1"); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("second accept error = %v, want conflict", err)
	}

	request := contract.FreezeContextPackRequest{
		ID:               "context-1",
		WorkItem:         contract.NodeRef{Kind: "task", ID: "task-1"},
		WorkItemRevision: "1",
		TargetEntityIDs:  []string{"service-1"},
		References: []contract.ContextReference{{
			Kind: "change", ID: "change-1", Revision: "accepted", Checksum: "sha256:change-1",
		}},
		PolicyVersion: "phase1-exit-v1",
	}
	if _, err := service.FreezeContextPack(ctx, member, "workspace-a", request); !errors.Is(err, contract.ErrForbidden) {
		t.Fatalf("member freeze error = %v, want forbidden", err)
	}
	pack, err := service.FreezeContextPack(ctx, owner, "workspace-a", request)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Checksum == "" || pack.WorkItem.ID != "task-1" || pack.WorkItemRevision != "1" || pack.PolicyVersion != "phase1-exit-v1" {
		t.Fatalf("pack = %+v", pack)
	}
	replayed, err := service.FreezeContextPack(ctx, owner, "workspace-a", request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Checksum != pack.Checksum || !replayed.CreatedAt.Equal(pack.CreatedAt) {
		t.Fatalf("replayed pack = %+v, want checksum=%s created_at=%s", replayed, pack.Checksum, pack.CreatedAt)
	}

	changed := request
	changed.WorkItemRevision = "2"
	if _, err := service.FreezeContextPack(ctx, owner, "workspace-a", changed); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("mutated frozen pack error = %v, want conflict", err)
	}
	missing := request
	missing.ID = "context-missing"
	missing.TargetEntityIDs = []string{"missing"}
	if _, err := service.FreezeContextPack(ctx, owner, "workspace-a", missing); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("missing target error = %v, want not found", err)
	}

	read, err := service.GetContextPack(ctx, member, "workspace-a", "context-1")
	if err != nil {
		t.Fatal(err)
	}
	if read.Checksum != pack.Checksum || len(read.References) != 1 || read.References[0].Revision != "accepted" {
		t.Fatalf("readback = %+v", read)
	}
}
