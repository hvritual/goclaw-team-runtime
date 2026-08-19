package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workspace "github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

func TestProjectSurfaceRepositoryCreatesProjectAndResourcesAtomicallyAndProjectsActiveCount(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := "2026-08-19T06:00:00Z"
	project := contract.ProjectSurfaceProject{
		ID: "project-atomic", WorkspaceID: "workspace-1", Title: "Atomic Project", Status: "planned", Priority: "none", CreatedAt: now, UpdatedAt: now,
	}
	resources := []application.ProjectResourceSeed{
		{ID: "resource-code", ResourceType: "github_repo", ResourceRef: contract.ProjectResourceRef{URL: "https://github.com/acme/runtime", Ref: "main"}, Fingerprint: strings.Repeat("a", 64), Label: "Code"},
		{ID: "resource-docs", ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://docs.example.com/runtime"}, Fingerprint: strings.Repeat("b", 64), Label: "Docs"},
	}
	created, err := repository.CreateProjectWithResources(context.Background(), project, resources, contract.WorkspaceActor{Type: "member", ID: "owner-1"})
	if err != nil {
		t.Fatalf("CreateProjectWithResources() error = %v", err)
	}
	if created.ResourceCount != 2 {
		t.Fatalf("created ResourceCount = %d", created.ResourceCount)
	}
	detail, err := repository.GetProject(context.Background(), "workspace-1", "project-atomic")
	if err != nil || detail.ResourceCount != 2 {
		t.Fatalf("detail = %#v, %v", detail, err)
	}
	listed, err := repository.ListProjects(context.Background(), "workspace-1", "")
	if err != nil {
		t.Fatal(err)
	}
	assertProjectResourceProjectionCount(t, listed, "project-atomic", 2)
	searched, total, err := repository.SearchProjects(context.Background(), application.ProjectSurfaceSearchQuery{
		WorkspaceID: "workspace-1", Phrase: "atomic", Terms: []string{"atomic"}, Limit: 20,
	})
	if err != nil || total != 1 || len(searched) != 1 || searched[0].Project.ResourceCount != 2 {
		t.Fatalf("search = %#v total=%d err=%v", searched, total, err)
	}

	resourcesRepository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	list, err := resourcesRepository.ListProjectResources(context.Background(), "workspace-1", "project-atomic", false)
	if err != nil || list.Total != 2 || list.Revision != 1 {
		t.Fatalf("resources = %#v, %v", list, err)
	}
	if err = resourcesRepository.ArchiveProjectResource(context.Background(), "workspace-1", "project-atomic", "resource-docs", 1, contract.WorkspaceActor{Type: "member", ID: "owner-1"}, time.Date(2026, 8, 19, 6, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	detail, err = repository.GetProject(context.Background(), "workspace-1", "project-atomic")
	if err != nil || detail.ResourceCount != 1 {
		t.Fatalf("detail after archive = %#v, %v", detail, err)
	}
}

func TestProjectSurfaceRepositoryRollsBackProjectWhenInitialResourceFails(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	project := contract.ProjectSurfaceProject{ID: "project-rollback", WorkspaceID: "workspace-1", Title: "Rollback", Status: "planned", Priority: "none", CreatedAt: "2026-08-19T06:00:00Z", UpdatedAt: "2026-08-19T06:00:00Z"}
	duplicate := strings.Repeat("c", 64)
	_, err = repository.CreateProjectWithResources(context.Background(), project, []application.ProjectResourceSeed{
		{ID: "resource-first", ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://one.example.com"}, Fingerprint: duplicate},
		{ID: "resource-second", ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://two.example.com"}, Fingerprint: duplicate},
	}, contract.WorkspaceActor{Type: "member", ID: "owner-1"})
	if err == nil {
		t.Fatal("CreateProjectWithResources() error = nil")
	}
	for name, query := range map[string]string{
		"project":  `SELECT COUNT(*) FROM workspace_projects WHERE id='project-rollback'`,
		"resource": `SELECT COUNT(*) FROM workspace_project_resources WHERE project_id='project-rollback'`,
		"set":      `SELECT COUNT(*) FROM workspace_project_resource_sets WHERE project_id='project-rollback'`,
		"audit":    `SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_id IN ('resource-first','resource-second')`,
	} {
		assertProjectResourceQueryCount(t, db, name, query, 0)
	}
}

func TestProjectSurfaceRepositoryRevalidatesInitialResourceManagerAndRollsBack(t *testing.T) {
	db := openProjectResourceDB(t)
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE auth_members SET role='member' WHERE workspace_id='workspace-1' AND user_id='owner-1'`); err != nil {
		t.Fatal(err)
	}
	leadType, leadID := "member", "different-member"
	project := contract.ProjectSurfaceProject{
		ID: "project-denied", WorkspaceID: "workspace-1", Title: "Denied Project", Status: "planned", Priority: "none",
		LeadType: &leadType, LeadID: &leadID, CreatedAt: "2026-08-19T06:00:00Z", UpdatedAt: "2026-08-19T06:00:00Z",
	}
	_, err = repository.CreateProjectWithResources(context.Background(), project, []application.ProjectResourceSeed{{
		ID: "resource-denied", ResourceType: "url", ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/denied"}, Fingerprint: strings.Repeat("f", 64),
	}}, contract.WorkspaceActor{Type: "member", ID: "owner-1"})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("CreateProjectWithResources() error = %v", err)
	}
	for name, query := range map[string]string{
		"project":  `SELECT COUNT(*) FROM workspace_projects WHERE id='project-denied'`,
		"resource": `SELECT COUNT(*) FROM workspace_project_resources WHERE project_id='project-denied'`,
		"set":      `SELECT COUNT(*) FROM workspace_project_resource_sets WHERE project_id='project-denied'`,
	} {
		assertProjectResourceQueryCount(t, db, name, query, 0)
	}
}

func TestBothProjectDeletionRepositoriesCleanLocalResourceAuthority(t *testing.T) {
	for _, useSurface := range []bool{true, false} {
		name := "generated project repository"
		if useSurface {
			name = "HTTP project surface repository"
		}
		t.Run(name, func(t *testing.T) {
			db := openProjectResourceDB(t)
			projectID := "project-delete"
			if _, err := db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at) VALUES(?, 'workspace-1','Delete Me','planned','none','[]','2026-08-19T06:00:00Z','2026-08-19T06:00:00Z')`, projectID); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO workspace_project_resource_sets(workspace_id,project_id,revision,updated_at) VALUES('workspace-1',?,1,'2026-08-19T06:00:00Z')`, projectID); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO workspace_project_resources(id,workspace_id,project_id,resource_type,canonical_url,fingerprint,position,status,revision,created_at,created_by,updated_at,updated_by) VALUES('resource-delete','workspace-1',?,'url','https://example.com',?,0,'active',1,'2026-08-19T06:00:00Z','owner-1','2026-08-19T06:00:00Z','owner-1')`, projectID, strings.Repeat("d", 64)); err != nil {
				t.Fatal(err)
			}
			if useSurface {
				repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
				if err != nil {
					t.Fatal(err)
				}
				if err = repository.DeleteProject(context.Background(), "workspace-1", projectID, time.Date(2026, 8, 19, 6, 2, 0, 0, time.UTC)); err != nil {
					t.Fatal(err)
				}
			} else {
				repository, err := persistence.NewProjectRepository(persistence.Config{DB: db})
				if err != nil {
					t.Fatal(err)
				}
				if err = repository.DeleteWithDependents(context.Background(), "workspace-1", projectID, time.Date(2026, 8, 19, 6, 2, 0, 0, time.UTC)); err != nil {
					t.Fatal(err)
				}
			}
			assertProjectResourceQueryCount(t, db, "project", `SELECT COUNT(*) FROM workspace_projects WHERE id='project-delete'`, 0)
			assertProjectResourceQueryCount(t, db, "resource", `SELECT COUNT(*) FROM workspace_project_resources WHERE project_id='project-delete'`, 0)
			assertProjectResourceQueryCount(t, db, "set", `SELECT COUNT(*) FROM workspace_project_resource_sets WHERE project_id='project-delete'`, 0)
		})
	}
}

func TestProjectResourceRepositorySurvivesSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project-resource-restart.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err = db.Exec(`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES('workspace-1','Workspace','workspace','WSP','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('owner-member','workspace-1','owner-1','owner','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at) VALUES('project-1','workspace-1','Runtime','planned','none','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
		ID: "resource-restart", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
		ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/restart"}, Fingerprint: strings.Repeat("e", 64),
		Label: "Restart", IdempotencyKey: "restart-1", RequestHash: strings.Repeat("5", 64),
		Actor: contract.WorkspaceActor{Type: "member", ID: "owner-1"}, OccurredAt: time.Date(2026, 8, 19, 6, 3, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	repository, err = persistence.NewProjectResourceRepository(persistence.Config{DB: reopened})
	if err != nil {
		t.Fatal(err)
	}
	list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
	if err != nil || list.Total != 1 || list.Revision != 1 || list.Resources[0].ID != "resource-restart" || list.Resources[0].Label != "Restart" {
		t.Fatalf("after restart = %#v, %v", list, err)
	}
}

func assertProjectResourceProjectionCount(t *testing.T, projects []contract.ProjectSurfaceProject, id string, want int) {
	t.Helper()
	for _, project := range projects {
		if project.ID == id {
			if project.ResourceCount != want {
				t.Fatalf("project %s ResourceCount = %d, want %d", id, project.ResourceCount, want)
			}
			return
		}
	}
	t.Fatalf("project %s not found in %#v", id, projects)
}

func assertProjectResourceQueryCount(t *testing.T, db *sql.DB, name, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", name, got, want)
	}
}
