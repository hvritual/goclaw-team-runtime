package workspace

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func seedProject(t *testing.T, db *sql.DB, id, workspaceID, name, description, status, assetIDs, createdAt, updatedAt string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspace_projects(
		id, workspace_id, name, description, status, asset_ids, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, workspaceID, name, description, status, assetIDs, createdAt, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSqliteProjectListAndSearchAreWorkspaceScopedAndDeterministic(t *testing.T) {
	ctx := context.Background()
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedWorkspace(t, db, "workspace-2", "Globex", "globex")
	seedProject(t, db, "p-exact", "workspace-1", "Delivery Plan", "", "planned", "[]", "2026-08-03T01:00:00Z", "2026-08-03T01:00:00Z")
	seedProject(t, db, "p-prefix", "workspace-1", "Delivery Plan Phase", "", "in_progress", "[]", "2026-08-03T02:00:00Z", "2026-08-03T02:00:00Z")
	seedProject(t, db, "p-phrase", "workspace-1", "Migration Delivery Plan", "", "paused", "[]", "2026-08-03T03:00:00Z", "2026-08-03T03:00:00Z")
	seedProject(t, db, "p-words", "workspace-1", "Plan for Delivery", "", "planned", "[]", "2026-08-03T04:00:00Z", "2026-08-03T04:00:00Z")
	seedProject(t, db, "p-description", "workspace-1", "Unrelated", "delivery plan in description", "planned", "[]", "2026-08-03T05:00:00Z", "2026-08-03T05:00:00Z")
	seedProject(t, db, "p-closed", "workspace-1", "Delivery Plan Archive", "", "completed", "[]", "2026-08-03T06:00:00Z", "2026-08-03T06:00:00Z")
	seedProject(t, db, "p-foreign", "workspace-2", "Delivery Plan", "", "planned", "[]", "2026-08-03T07:00:00Z", "2026-08-03T07:00:00Z")
	module := newWorkspaceServicesTestModule(t, db, &workspaceActorCatalog{}, "unused")

	listed, err := module.ProjectLocal().ListProjects(ctx, contract.ListProjectsRequest{WorkspaceId: "workspace-1", Status: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if listed.Total != 3 || len(listed.Projects) != 3 {
		t.Fatalf("ListProjects() total/projects = %d/%d", listed.Total, len(listed.Projects))
	}
	wantList := []string{"p-description", "p-words", "p-exact"}
	for index, want := range wantList {
		if listed.Projects[index].Id != want {
			t.Fatalf("ListProjects() IDs = %+v", listed.Projects)
		}
	}

	searched, err := module.ProjectLocal().SearchProjects(ctx, contract.SearchProjectsRequest{
		WorkspaceId: "workspace-1", Query: "  DELIVERY PLAN  ", Limit: 2, Offset: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if searched.Total != 5 || len(searched.Hits) != 2 || searched.Hits[0].Project.Id != "p-prefix" || searched.Hits[1].Project.Id != "p-phrase" {
		t.Fatalf("SearchProjects() = %+v", searched)
	}
	if searched.Hits[0].MatchSource != "name" || searched.Hits[0].MatchedSnippet == nil {
		t.Fatalf("SearchProjects() match metadata = %+v", searched.Hits[0])
	}
	withClosed, err := module.ProjectLocal().SearchProjects(ctx, contract.SearchProjectsRequest{
		WorkspaceId: "workspace-1", Query: "delivery plan", IncludeClosed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if withClosed.Total != 6 || len(withClosed.Hits) != 6 || withClosed.Hits[0].Project.Id != "p-exact" || withClosed.Hits[1].Project.Id != "p-closed" {
		t.Fatalf("SearchProjects(include_closed) = %+v", withClosed)
	}
}

func TestSqliteProjectUpdateAndDeleteOwnInternalCleanup(t *testing.T) {
	ctx := context.Background()
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedProject(t, db, "p-delete", "workspace-1", "Delivery", "old", "planned", `["asset-1"]`, "2026-08-03T01:00:00Z", "2026-08-03T01:00:00Z")
	seedProject(t, db, "p-other", "workspace-1", "Other", "", "planned", "[]", "2026-08-03T01:00:00Z", "2026-08-03T01:00:00Z")
	_, err := db.Exec(`INSERT INTO workspace_project_actor_relations(workspace_id, project_id, actor_type, actor_id, role)
		VALUES ('workspace-1', 'p-delete', 'member', 'member-1', 'lead')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO workspace_todos(id, workspace_id, title, status, project_id, created_at, updated_at)
		VALUES ('todo-delete', 'workspace-1', 'Todo', 'todo', 'p-delete', '2026-08-03T01:00:00Z', '2026-08-03T01:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	seedWorkspaceIssue(t, db, "issue-delete", "workspace-1", "WSP-10", "p-delete")
	_, err = db.Exec(`INSERT INTO workspace_requirements(
		id, workspace_id, project_id, title, current_version, approval_status, coverage_status, issue_ids, created_at, updated_at
	) VALUES ('requirement-delete', 'workspace-1', 'p-delete', 'Requirement', 1, 'draft', 'uncovered', '[]', '2026-08-03T01:00:00Z', '2026-08-03T01:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO workspace_requirement_versions(id, requirement_id, version, content, created_at)
		VALUES ('requirement-version-delete', 'requirement-delete', 1, 'content', '2026-08-03T01:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	module := newWorkspaceServicesTestModule(t, db, &workspaceActorCatalog{}, "unused")
	name, description, status := "  Launch  ", "", "in_progress"
	updated, err := module.ProjectLocal().UpdateProject(ctx, contract.UpdateProjectRequest{
		WorkspaceId: "workspace-1", ProjectId: "p-delete", Name: &name, Description: &description, Status: &status,
	})
	if err != nil || updated.Project == nil {
		t.Fatalf("UpdateProject() = %+v, %v", updated.Project, err)
	}
	if updated.Project.Name != "Launch" || updated.Project.Description != "" || updated.Project.Status != "in_progress" || len(updated.Project.AssetIds) != 1 || updated.Project.AssetIds[0] != "asset-1" {
		t.Fatalf("updated Project = %+v", updated.Project)
	}
	if _, err := module.ProjectLocal().DeleteProject(ctx, contract.DeleteProjectRequest{WorkspaceId: "workspace-1", ProjectId: "p-delete"}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM workspace_projects WHERE id = 'p-delete'`,
		`SELECT COUNT(*) FROM workspace_project_actor_relations WHERE project_id = 'p-delete'`,
		`SELECT COUNT(*) FROM workspace_requirements WHERE project_id = 'p-delete'`,
		`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id = 'requirement-delete'`,
	} {
		var count int
		if err := db.QueryRow(query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("cleanup query %q count/error = %d/%v", query, count, err)
		}
	}
	for _, table := range []string{"workspace_todos", "workspace_issues"} {
		var projectID sql.NullString
		var updatedAt string
		if err := db.QueryRow(`SELECT project_id, updated_at FROM `+table+` WHERE id = ?`, map[string]string{"workspace_todos": "todo-delete", "workspace_issues": "issue-delete"}[table]).Scan(&projectID, &updatedAt); err != nil {
			t.Fatal(err)
		}
		if projectID.Valid || updatedAt != "2026-08-03T04:05:06.000000007Z" {
			t.Fatalf("%s project_id/updated_at = %+v/%q", table, projectID, updatedAt)
		}
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_projects WHERE id = 'p-other'`).Scan(&remaining); err != nil || remaining != 1 {
		t.Fatalf("unrelated Project count/error = %d/%v", remaining, err)
	}
	if _, err := module.ProjectLocal().DeleteProject(ctx, contract.DeleteProjectRequest{WorkspaceId: "workspace-1", ProjectId: "p-delete"}); !errors.Is(err, contract.ErrProjectNotFound) {
		t.Fatalf("second DeleteProject() error = %v", err)
	}
}

func TestSqliteProjectDeleteRollsBackEveryCleanupOnFailure(t *testing.T) {
	ctx := context.Background()
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedProject(t, db, "p-rollback", "workspace-1", "Rollback", "", "planned", "[]", "2026-08-03T01:00:00Z", "2026-08-03T01:00:00Z")
	_, err := db.Exec(`INSERT INTO workspace_project_actor_relations(workspace_id, project_id, actor_type, actor_id, role)
		VALUES ('workspace-1', 'p-rollback', 'member', 'member-1', 'lead')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO workspace_todos(id, workspace_id, title, status, project_id, created_at, updated_at)
		VALUES ('todo-rollback', 'workspace-1', 'Todo', 'todo', 'p-rollback', '2026-08-03T01:00:00Z', '2026-08-03T01:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TRIGGER reject_project_delete BEFORE DELETE ON workspace_projects
		WHEN OLD.id = 'p-rollback' BEGIN SELECT RAISE(ABORT, 'reject test deletion'); END`)
	if err != nil {
		t.Fatal(err)
	}
	module := newWorkspaceServicesTestModule(t, db, &workspaceActorCatalog{}, "unused")
	if _, err := module.ProjectLocal().DeleteProject(ctx, contract.DeleteProjectRequest{WorkspaceId: "workspace-1", ProjectId: "p-rollback"}); err == nil {
		t.Fatal("DeleteProject() error = nil")
	}
	var projectCount, relationCount int
	var todoProjectID string
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_projects WHERE id = 'p-rollback'`).Scan(&projectCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_project_actor_relations WHERE project_id = 'p-rollback'`).Scan(&relationCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT project_id FROM workspace_todos WHERE id = 'todo-rollback'`).Scan(&todoProjectID); err != nil {
		t.Fatal(err)
	}
	if projectCount != 1 || relationCount != 1 || todoProjectID != "p-rollback" {
		t.Fatalf("rollback state = project:%d relation:%d todo.project:%q", projectCount, relationCount, todoProjectID)
	}
}

func TestProjectLifecycleClockIsUTC(t *testing.T) {
	// Guard the exact cleanup timestamp asserted above against accidental fixture drift.
	if got := time.Date(2026, 8, 3, 4, 5, 6, 7, time.UTC).Format(time.RFC3339Nano); got != "2026-08-03T04:05:06.000000007Z" {
		t.Fatalf("fixture timestamp = %q", got)
	}
}
