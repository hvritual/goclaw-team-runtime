package application

import (
	"context"
	"errors"
	"testing"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

func TestWorkLinksDeriveCanonicalRelationsAndProvenance(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{
		ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}

	project, err := service.PutWorkLink(ctx, contract.PutWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1", EntityID: "service-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.Relation != "changes" || project.Authority != "authoritative" || project.Provenance.SourceType != "workspace" {
		t.Fatalf("project link = %+v", project)
	}
	if project.Provenance.Locator != "workspace://workspace-a/work/project/project-1" || project.Provenance.ObservedAt.IsZero() {
		t.Fatalf("project provenance = %+v", project.Provenance)
	}
	again, err := service.PutWorkLink(ctx, contract.PutWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1", EntityID: "service-1",
	})
	if err != nil || again.ID != project.ID {
		t.Fatalf("idempotent link = %+v err=%v", again, err)
	}

	task, err := service.PutWorkLink(ctx, contract.PutWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkTask, WorkID: "task-1", EntityID: "service-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Relation != "affects" {
		t.Fatalf("task relation = %q", task.Relation)
	}

	links, err := service.ListWorkLinks(ctx, contract.ListWorkLinksRequest{WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1"})
	if err != nil || len(links) != 1 || links[0].ID != project.ID {
		t.Fatalf("project links = %+v err=%v", links, err)
	}
	isolated, err := service.ListWorkLinks(ctx, contract.ListWorkLinksRequest{WorkspaceID: "workspace-b", WorkKind: contract.WorkLinkProject, WorkID: "project-1"})
	if err != nil || len(isolated) != 0 {
		t.Fatalf("cross-workspace links = %+v err=%v", isolated, err)
	}
}

func TestWorkLinksRetainHistoryAcrossArchiveAndRequireExplicitUnlink(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{
		ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PutWorkLink(ctx, contract.PutWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1", EntityID: "service-1",
	}); err != nil {
		t.Fatal(err)
	}

	archived := "archived"
	if _, err := service.UpdateEntity(ctx, owner, "workspace-a", "service-1", contract.UpdateEntityRequest{Status: &archived}); err != nil {
		t.Fatal(err)
	}
	links, err := service.ListWorkLinks(ctx, contract.ListWorkLinksRequest{WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1"})
	if err != nil || len(links) != 1 {
		t.Fatalf("links after archive = %+v err=%v", links, err)
	}
	if _, err := service.PutWorkLink(ctx, contract.PutWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkRequirement, WorkID: "requirement-1", EntityID: "service-1",
	}); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("link archived entity error = %v, want conflict", err)
	}
	if err := service.DeleteWorkLink(ctx, contract.DeleteWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1", EntityID: "service-1",
	}); err != nil {
		t.Fatal(err)
	}
	links, err = service.ListWorkLinks(ctx, contract.ListWorkLinksRequest{WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1"})
	if err != nil || len(links) != 0 {
		t.Fatalf("links after explicit unlink = %+v err=%v", links, err)
	}
}

func TestWorkLinksRejectMissingEntity(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	_, err := service.PutWorkLink(context.Background(), contract.PutWorkLinkRequest{
		WorkspaceID: "workspace-a", WorkKind: contract.WorkLinkProject, WorkID: "project-1", EntityID: "missing",
	})
	if !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("error = %v, want not found", err)
	}
}
