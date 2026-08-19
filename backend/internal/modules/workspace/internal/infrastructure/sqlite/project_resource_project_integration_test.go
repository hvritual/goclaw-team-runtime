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
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
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
			seedProjectRequirementDeletionAuthority(t, db, projectID)
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
			for _, table := range []string{
				"workspace_requirement_baselines", "workspace_requirement_revisions", "workspace_requirement_issue_links",
				"workspace_requirement_outline_links", "workspace_requirement_review_projections",
				"workspace_project_requirement_access_sets", "workspace_project_requirement_grants",
				"workspace_project_outline_sets", "workspace_project_outline_nodes",
				"workspace_requirements", "workspace_requirement_versions",
				"workspace_audit_entries", "workspace_mutation_idempotency", "workspace_outbox_events",
			} {
				assertProjectResourceQueryCount(t, db, table, `SELECT COUNT(*) FROM `+table, 0)
			}
		})
	}
}

func seedProjectRequirementDeletionAuthority(t *testing.T, db *sql.DB, projectID string) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('editor-member','workspace-1','editor-1','member','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at)
		 VALUES('issue-delete','workspace-1',77,'ONE-77','Delete-linked Issue','todo','none','member','editor-1','project-delete','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at)
		 VALUES('legacy-delete','workspace-1','project-delete','Legacy',1,'draft','covered','["issue-delete"]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at)
		 VALUES('legacy-delete-v1','legacy-delete',1,'Legacy content','2026-08-19T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 6, 1, 0, 0, time.UTC)
	owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	editor := contract.WorkspaceActor{Type: "member", ID: "editor-1"}
	if _, err = repository.ReplaceProjectRequirementAccess(context.Background(), application.ProjectRequirementAccessReplace{
		WorkspaceID: "workspace-1", ProjectID: projectID, ExpectedRevision: 0,
		Grants: []application.ProjectRequirementGrantChange{{MemberID: "editor-member", GrantKind: "project_editor"}},
		Actor:  owner, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreateProjectOutlineNode(context.Background(), application.ProjectOutlineNodeCreate{
		NodeID: "outline-delete", WorkspaceID: "workspace-1", ProjectID: projectID, ExpectedRevision: 0,
		Title: "Delete root", IdempotencyKey: "outline-delete-key", RequestHash: strings.Repeat("8", 64),
		Actor: editor, OccurredAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-delete", WorkspaceID: "workspace-1", ProjectID: projectID, ExpectedRevision: 0,
		Content:       requirementDomain.Content{ProblemStatement: "Delete baseline", Goals: []requirementDomain.Item{{Key: "goal-delete", Text: "Delete safely"}}},
		ChangeSummary: "Initial", IdempotencyKey: "baseline-delete-key", RequestHash: strings.Repeat("9", 64),
		Actor: editor, OccurredAt: now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	mutate := func(kind, target string, expected int64) {
		t.Helper()
		if _, mutationErr := repository.MutateProjectRequirementLink(context.Background(), application.ProjectRequirementLinkMutation{
			WorkspaceID: "workspace-1", ProjectID: projectID, RequirementKey: "goal-delete", TargetKind: kind,
			TargetID: target, ExpectedRevision: expected, Actor: editor, OccurredAt: now.Add(time.Duration(expected+2) * time.Minute),
		}); mutationErr != nil {
			t.Fatal(mutationErr)
		}
	}
	mutate("outline", "outline-delete", 1)
	mutate("issue", "issue-delete", 2)
	transition := func(action string, expected int64, actor contract.WorkspaceActor) {
		t.Helper()
		if _, transitionErr := repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
			WorkspaceID: "workspace-1", ProjectID: projectID, Action: action, ExpectedRevision: expected,
			Actor: actor, OccurredAt: now.Add(time.Duration(expected+4) * time.Minute),
		}); transitionErr != nil {
			t.Fatal(transitionErr)
		}
	}
	transition("submit_review", 3, editor)
	transition("approve", 4, owner)
	transition("freeze", 5, owner)
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		WorkspaceID: "workspace-1", ProjectID: projectID, ExpectedRevision: 6,
		Content:       requirementDomain.Content{ProblemStatement: "Changed before delete", Goals: []requirementDomain.Item{{Key: "goal-delete", Text: "Delete safely"}}},
		ChangeSummary: "Material delete change", MaterialChange: true, Actor: editor, OccurredAt: now.Add(11 * time.Minute),
	}); err != nil {
		t.Fatal(err)
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
