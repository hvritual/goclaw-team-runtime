package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	workspace "github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestProjectResourceRepositoryLifecycleAndReplay(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}

	firstCommand := application.ProjectResourceCreate{
		ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
		ResourceType: "github_repo", ResourceRef: contract.ProjectResourceRef{URL: "https://github.com/acme/repo", Ref: "main"},
		Fingerprint: strings.Repeat("a", 64), Label: "Runtime", IdempotencyKey: "create-1",
		RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
	}
	first, err := repository.CreateProjectResource(context.Background(), firstCommand)
	if err != nil || first.ID != "resource-1" || first.Revision != 1 || first.Position != 0 {
		t.Fatalf("first create = %#v, %v", first, err)
	}
	replayCommand := firstCommand
	replayCommand.ID = "resource-replay-must-not-persist"
	replay, err := repository.CreateProjectResource(context.Background(), replayCommand)
	if err != nil || replay.ID != first.ID {
		t.Fatalf("replay = %#v, %v", replay, err)
	}
	conflictCommand := firstCommand
	conflictCommand.ID = "resource-conflict"
	conflictCommand.RequestHash = strings.Repeat("2", 64)
	if _, err = repository.CreateProjectResource(context.Background(), conflictCommand); !errors.Is(err, application.ErrProjectResourceConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}
	duplicateCommand := firstCommand
	duplicateCommand.ID = "resource-duplicate"
	duplicateCommand.IdempotencyKey = "create-duplicate"
	duplicateCommand.RequestHash = strings.Repeat("3", 64)
	if _, err = repository.CreateProjectResource(context.Background(), duplicateCommand); !errors.Is(err, application.ErrProjectResourceConflict) {
		t.Fatalf("duplicate error = %v", err)
	}

	secondCommand := application.ProjectResourceCreate{
		ID: "resource-2", WorkspaceID: "workspace-1", ProjectID: "project-1",
		ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
		Fingerprint: strings.Repeat("b", 64), Label: "Docs", IdempotencyKey: "create-2",
		RequestHash: strings.Repeat("4", 64), Actor: actor, OccurredAt: now.Add(time.Second),
	}
	second, err := repository.CreateProjectResource(context.Background(), secondCommand)
	if err != nil || second.Revision != 2 || second.Position != 1 {
		t.Fatalf("second create = %#v, %v", second, err)
	}
	duplicateReference := contract.ProjectResourceRef{URL: "https://github.com/acme/repo", Ref: "main"}
	if _, err = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-2",
		Action: "update", ExpectedRevision: 2, ResourceRef: &duplicateReference, Fingerprint: strings.Repeat("a", 64),
		Actor: actor, OccurredAt: now.Add(1500 * time.Millisecond),
	}); !errors.Is(err, application.ErrProjectResourceConflict) {
		t.Fatalf("duplicate update error = %v", err)
	}

	before := "resource-1"
	reordered, err := repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-2",
		Action: "reorder", ExpectedRevision: 2, BeforeResourceID: &before,
		Actor: actor, OccurredAt: now.Add(2 * time.Second),
	})
	if err != nil || reordered.Revision != 3 || reordered.Position != 0 {
		t.Fatalf("reorder = %#v, %v", reordered, err)
	}
	if _, err = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1",
		Action: "reorder", ExpectedRevision: 2, Actor: actor, OccurredAt: now.Add(3 * time.Second),
	}); !errors.Is(err, contract.ErrRevisionConflict) {
		t.Fatalf("stale reorder error = %v", err)
	}

	if err = repository.ArchiveProjectResource(context.Background(), "workspace-1", "project-1", "resource-1", 3, actor, now.Add(4*time.Second)); err != nil {
		t.Fatalf("archive = %v", err)
	}
	active, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
	if err != nil || len(active.Resources) != 1 || active.Resources[0].ID != "resource-2" || active.Revision != 4 {
		t.Fatalf("active list = %#v, %v", active, err)
	}
	archived, err := repository.GetProjectResource(context.Background(), "workspace-1", "project-1", "resource-1")
	if err != nil || archived.Status != "archived" || archived.ArchivedBy != "owner-1" {
		t.Fatalf("archived = %#v, %v", archived, err)
	}
	restored, err := repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1",
		Action: "restore", ExpectedRevision: 4, Actor: actor, OccurredAt: now.Add(5 * time.Second),
	})
	if err != nil || restored.Status != "active" || restored.Revision != 5 || restored.Position != 1 {
		t.Fatalf("restore = %#v, %v", restored, err)
	}

	assertProjectResourceCount(t, db, "workspace_project_resources", 2)
	assertProjectResourceCount(t, db, "workspace_mutation_idempotency", 2)
	var audits int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='project_resource'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 5 {
		t.Fatalf("audit count = %d, want 5", audits)
	}
}

func TestProjectResourceRepositoryHidesForeignProject(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ProjectResourceAccess(context.Background(), "workspace-2", "project-1"); !errors.Is(err, application.ErrProjectSurfaceNotFound) {
		t.Fatalf("foreign access error = %v", err)
	}
	if _, err = repository.GetProjectResource(context.Background(), "workspace-2", "project-1", "resource-1"); !errors.Is(err, application.ErrProjectResourceNotFound) {
		t.Fatalf("foreign resource error = %v", err)
	}
}

func TestProjectResourceRepositoryRevalidatesCurrentManagerInsideWriteTransaction(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE auth_members SET role='member' WHERE workspace_id='workspace-1' AND user_id='owner-1'`); err != nil {
		t.Fatal(err)
	}
	_, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
		ID: "resource-denied", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
		ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/denied"}, Fingerprint: strings.Repeat("d", 64),
		IdempotencyKey: "denied-1", RequestHash: strings.Repeat("9", 64),
		Actor: contract.WorkspaceActor{Type: "member", ID: "owner-1"}, OccurredAt: time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("CreateProjectResource() error = %v", err)
	}
	assertProjectResourceCount(t, db, "workspace_project_resources", 0)
	assertProjectResourceCount(t, db, "workspace_project_resource_sets", 0)
	assertProjectResourceCount(t, db, "workspace_mutation_idempotency", 0)
}

func TestProjectResourceRepositoryRejectsInvalidStatusTransitionsWithoutAdvancingSet(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	command := application.ProjectResourceCreate{
		ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
		ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
		Fingerprint: strings.Repeat("a", 64), IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
	}
	if _, err = repository.CreateProjectResource(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if err = repository.ArchiveProjectResource(context.Background(), "workspace-1", "project-1", "resource-1", 1, actor, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	label := "must not update"
	if _, err = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "update", ExpectedRevision: 2,
		Label: &label, Actor: actor, OccurredAt: now.Add(2 * time.Second),
	}); !errors.Is(err, application.ErrInvalidProjectResourceRequest) {
		t.Fatalf("archived update error = %v", err)
	}
	list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if list.Revision != 2 || len(list.Resources) != 1 || list.Resources[0].Revision != 2 || list.Resources[0].Label != "" {
		t.Fatalf("list after invalid transition = %#v", list)
	}
}

func TestProjectResourceRepositoryConcurrentMutationHasOneWinnerAndOneStaleConflict(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
		ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
		ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Fingerprint: strings.Repeat("a", 64),
		IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errorsByWriter := make([]error, 2)
	var wait sync.WaitGroup
	for index, label := range []string{"Writer A", "Writer B"} {
		wait.Add(1)
		go func(index int, label string) {
			defer wait.Done()
			<-start
			_, errorsByWriter[index] = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
				WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "update", ExpectedRevision: 1,
				Label: &label, Actor: actor, OccurredAt: now.Add(time.Duration(index+1) * time.Second),
			})
		}(index, label)
	}
	close(start)
	wait.Wait()
	winners, conflicts := 0, 0
	for _, mutationErr := range errorsByWriter {
		switch {
		case mutationErr == nil:
			winners++
		case errors.Is(mutationErr, contract.ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected mutation error = %v", mutationErr)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d errors=%v", winners, conflicts, errorsByWriter)
	}
	list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
	if err != nil || list.Revision != 2 || len(list.Resources) != 1 || list.Resources[0].Revision != 2 {
		t.Fatalf("list = %#v, %v", list, err)
	}
	var audits int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_id='resource-1'`).Scan(&audits); err != nil || audits != 2 {
		t.Fatalf("audit count = %d, err=%v", audits, err)
	}
}

func TestProjectResourceRepositoryConcurrentDuplicateCreateHasOneWinner(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	start := make(chan struct{})
	errorsByWriter := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsByWriter {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsByWriter[index] = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
				ID: "resource-" + strconv.Itoa(index+1), WorkspaceID: "workspace-1", ProjectID: "project-1",
				ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"},
				Fingerprint: strings.Repeat("f", 64), IdempotencyKey: "duplicate-" + strconv.Itoa(index+1),
				RequestHash: strings.Repeat(strconv.Itoa(index+1), 64), Actor: actor,
				OccurredAt: now.Add(time.Duration(index) * time.Second),
			})
		}(index)
	}
	close(start)
	wait.Wait()
	winners, conflicts := 0, 0
	for _, createErr := range errorsByWriter {
		switch {
		case createErr == nil:
			winners++
		case errors.Is(createErr, application.ErrProjectResourceConflict):
			conflicts++
		default:
			t.Fatalf("unexpected create error = %v", createErr)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d errors=%v", winners, conflicts, errorsByWriter)
	}
	list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
	if err != nil || list.Total != 1 || list.Revision != 1 {
		t.Fatalf("list = %#v, %v", list, err)
	}
	assertProjectResourceCount(t, db, "workspace_mutation_idempotency", 1)
	var audits int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='project_resource'`).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit count = %d, err=%v", audits, err)
	}
}

func TestProjectResourceRepositoryRevisionPrecedesStateAndSuppressesStaleRefresh(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	for index := 1; index <= 2; index++ {
		if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
			ID: "resource-" + strconv.Itoa(index), WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
			ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/" + strconv.Itoa(index)}, Fingerprint: strings.Repeat(strconv.Itoa(index), 64),
			IdempotencyKey: "create-" + strconv.Itoa(index), RequestHash: strings.Repeat(strconv.Itoa(index+2), 64), Actor: actor, OccurredAt: now.Add(time.Duration(index) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	_, err = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "restore",
		ExpectedRevision: 1, Actor: actor, OccurredAt: now.Add(3 * time.Second),
	})
	var restoreConflict contract.RevisionConflictError
	if !errors.As(err, &restoreConflict) || restoreConflict.CurrentRevision != 2 {
		t.Fatalf("stale restore error = %v", err)
	}
	if err = repository.ArchiveProjectResource(context.Background(), "workspace-1", "project-1", "resource-1", 2, actor, now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	resolverCalls := 0
	_, err = repository.RefreshProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "refresh",
		ExpectedRevision: 2, Actor: actor, OccurredAt: now.Add(5 * time.Second),
	}, func(context.Context, contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection {
		resolverCalls++
		return contract.ProjectResourceConnection{State: "available", CheckedAt: now.Format(time.RFC3339Nano)}
	})
	var refreshConflict contract.RevisionConflictError
	if !errors.As(err, &refreshConflict) || refreshConflict.CurrentRevision != 3 {
		t.Fatalf("stale refresh error = %v", err)
	}
	if resolverCalls != 0 {
		t.Fatalf("stale refresh resolver calls = %d", resolverCalls)
	}
}

func TestProjectResourceRepositoryConcurrentRefreshInvokesResolverOnlyForWinner(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
		ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
		ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Fingerprint: strings.Repeat("a", 64),
		IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var resolverCalls atomic.Int32
	errorsByWriter := make([]error, 2)
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		_, errorsByWriter[0] = repository.RefreshProjectResource(context.Background(), application.ProjectResourceMutation{
			WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "refresh",
			ExpectedRevision: 1, Actor: actor, OccurredAt: now.Add(time.Second),
		}, func(context.Context, contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection {
			resolverCalls.Add(1)
			close(entered)
			<-release
			return contract.ProjectResourceConnection{State: "available", CheckedAt: now.Add(time.Second).Format(time.RFC3339Nano)}
		})
	}()
	<-entered
	secondStarted, secondDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(secondDone)
		close(secondStarted)
		_, errorsByWriter[1] = repository.RefreshProjectResource(context.Background(), application.ProjectResourceMutation{
			WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "refresh",
			ExpectedRevision: 1, Actor: actor, OccurredAt: now.Add(2 * time.Second),
		}, func(context.Context, contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection {
			resolverCalls.Add(1)
			return contract.ProjectResourceConnection{State: "available", CheckedAt: now.Add(2 * time.Second).Format(time.RFC3339Nano)}
		})
	}()
	<-secondStarted
	close(release)
	<-firstDone
	<-secondDone
	if errorsByWriter[0] != nil || !errors.Is(errorsByWriter[1], contract.ErrRevisionConflict) {
		t.Fatalf("refresh errors = %v", errorsByWriter)
	}
	if resolverCalls.Load() != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolverCalls.Load())
	}
	list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
	if err != nil || list.Revision != 2 || list.Resources[0].Revision != 2 || list.Resources[0].Connection.State != "available" {
		t.Fatalf("list = %#v, %v", list, err)
	}
}

func TestProjectResourceRepositoryCurrentAuthorityDeniesEveryMutationBeforeEffects(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
		ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
		ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Fingerprint: strings.Repeat("a", 64),
		IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE auth_members SET role='member' WHERE workspace_id='workspace-1' AND user_id='owner-1'`); err != nil {
		t.Fatal(err)
	}
	label := "Denied"
	if _, err = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "update",
		ExpectedRevision: 1, Label: &label, Actor: actor, OccurredAt: now.Add(time.Second),
	}); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("update error = %v", err)
	}
	resolverCalls := 0
	if _, err = repository.RefreshProjectResource(context.Background(), application.ProjectResourceMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "refresh",
		ExpectedRevision: 1, Actor: actor, OccurredAt: now.Add(2 * time.Second),
	}, func(context.Context, contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection {
		resolverCalls++
		return contract.ProjectResourceConnection{State: "available"}
	}); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("refresh error = %v", err)
	}
	if err = repository.ArchiveProjectResource(context.Background(), "workspace-1", "project-1", "resource-1", 1, actor, now.Add(3*time.Second)); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("archive error = %v", err)
	}
	if resolverCalls != 0 {
		t.Fatalf("resolver calls = %d", resolverCalls)
	}
	list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
	if err != nil || list.Revision != 1 || list.Resources[0].Label != "" || list.Resources[0].Status != "active" {
		t.Fatalf("list = %#v, %v", list, err)
	}
	var audits int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='project_resource'`).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit count = %d, err=%v", audits, err)
	}
}

func TestProjectResourceRepositoryRevalidatesCurrentLeadAndProjectStatus(t *testing.T) {
	t.Run("lead changed", func(t *testing.T) {
		db := openProjectResourceDB(t)
		if _, err := db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('lead-member','workspace-1','lead-user','member','2026-08-19T00:00:00Z')`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE workspace_projects SET lead_type='member',lead_id='lead-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
			t.Fatal(err)
		}
		repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
		actor := contract.WorkspaceActor{Type: "member", ID: "lead-user"}
		if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
			ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
			ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Fingerprint: strings.Repeat("a", 64),
			IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`UPDATE workspace_projects SET lead_id='another-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
			t.Fatal(err)
		}
		label := "Denied after lead change"
		_, err = repository.MutateProjectResource(context.Background(), application.ProjectResourceMutation{
			WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "update",
			ExpectedRevision: 1, Label: &label, Actor: actor, OccurredAt: now.Add(time.Second),
		})
		if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
			t.Fatalf("update error = %v", err)
		}
	})

	t.Run("project completed", func(t *testing.T) {
		db := openProjectResourceDB(t)
		repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
		actor := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
		if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
			ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
			ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Fingerprint: strings.Repeat("a", 64),
			IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`UPDATE workspace_projects SET status='completed' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
			t.Fatal(err)
		}
		resolverCalls := 0
		_, err = repository.RefreshProjectResource(context.Background(), application.ProjectResourceMutation{
			WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceID: "resource-1", Action: "refresh",
			ExpectedRevision: 1, Actor: actor, OccurredAt: now.Add(time.Second),
		}, func(context.Context, contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection {
			resolverCalls++
			return contract.ProjectResourceConnection{State: "available"}
		})
		if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
			t.Fatalf("refresh error = %v", err)
		}
		if resolverCalls != 0 {
			t.Fatalf("resolver calls = %d", resolverCalls)
		}
	})
}

func openProjectResourceDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "project-resources.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE auth_members (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('owner','admin','member')),
		created_at TEXT NOT NULL,
		UNIQUE (workspace_id,user_id)
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES
		('workspace-1','Workspace One','workspace-one','ONE','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z'),
		('workspace-2','Workspace Two','workspace-two','TWO','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO workspace_projects(
		id,workspace_id,name,status,priority,lead_type,lead_id,asset_ids,created_at,updated_at
	) VALUES('project-1','workspace-1','Project One','in_progress','none','member','lead-1','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
		('owner-member','workspace-1','owner-1','owner','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func assertProjectResourceCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
