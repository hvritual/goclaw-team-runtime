package sqlite_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

func TestBothProjectDeletionRepositoriesCleanRetrospectivesAndRetainTargetsAndEvidence(t *testing.T) {
	for _, useSurface := range []bool{true, false} {
		name := "installed local project repository"
		if useSurface {
			name = "canonical HTTP project surface repository"
		}
		t.Run(name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			seedProjectRetrospectiveDeletionAuthority(t, db)
			deleteProjectThroughInstalledRepository(t, db, useSurface)

			for _, table := range []string{
				"workspace_project_retrospectives",
				"workspace_project_retrospective_revisions",
				"workspace_project_retrospective_participants",
				"workspace_project_retrospective_action_links",
			} {
				assertProjectRetrospectiveProjectCount(t, db, table+" owning project", `SELECT COUNT(*) FROM `+table+` WHERE workspace_id='workspace-1' AND project_id='project-1'`, 0)
				assertProjectRetrospectiveProjectCount(t, db, table+" foreign project", `SELECT COUNT(*) FROM `+table+` WHERE workspace_id='workspace-1' AND project_id='project-2'`, 1)
			}
			assertProjectRetrospectiveProjectCount(t, db, "resource revision", `SELECT COUNT(*) FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='project_retrospective' AND resource_id='retro-delete'`, 0)
			assertProjectRetrospectiveProjectCount(t, db, "owning project", `SELECT COUNT(*) FROM workspace_projects WHERE workspace_id='workspace-1' AND id='project-1'`, 0)
			assertProjectRetrospectiveProjectCount(t, db, "foreign project", `SELECT COUNT(*) FROM workspace_projects WHERE workspace_id='workspace-1' AND id='project-2'`, 1)
			assertProjectRetrospectiveProjectCount(t, db, "linked issue", `SELECT COUNT(*) FROM workspace_issues WHERE workspace_id='workspace-1' AND id='issue-retro-target'`, 1)
			var issueProject sql.NullString
			if err := db.QueryRow(`SELECT project_id FROM workspace_issues WHERE workspace_id='workspace-1' AND id='issue-retro-target'`).Scan(&issueProject); err != nil || issueProject.Valid {
				t.Fatalf("linked Issue project = %#v, error %v", issueProject, err)
			}
			assertProjectRetrospectiveProjectCount(t, db, "retained audit", `SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='project_retrospective' AND resource_id='retro-delete'`, 2)
			assertProjectRetrospectiveProjectCount(t, db, "retained outbox", `SELECT COUNT(*) FROM workspace_outbox_events WHERE aggregate_kind='project_retrospective' AND aggregate_id='retro-delete'`, 2)
			assertProjectRetrospectiveProjectCount(t, db, "retained create replay", `SELECT COUNT(*) FROM workspace_mutation_idempotency WHERE resource_kind='project_retrospective' AND resource_id='retro-delete'`, 1)
		})
	}
}

func TestBothProjectDeletionRepositoriesRollBackRetrospectiveCleanupOnLaterFailure(t *testing.T) {
	for _, useSurface := range []bool{true, false} {
		name := "installed local project repository"
		if useSurface {
			name = "canonical HTTP project surface repository"
		}
		t.Run(name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			seedProjectRetrospectiveDeletionAuthority(t, db)
			if _, err := db.Exec(`CREATE TRIGGER force_project_delete_failure BEFORE DELETE ON workspace_projects
				WHEN OLD.workspace_id='workspace-1' AND OLD.id='project-1'
				BEGIN SELECT RAISE(ABORT,'forced project delete failure'); END`); err != nil {
				t.Fatal(err)
			}
			if err := deleteProjectThroughInstalledRepositoryResult(db, useSurface); err == nil {
				t.Fatal("forced Project deletion error = nil")
			}
			for table, want := range map[string]int{
				"workspace_project_retrospectives":             1,
				"workspace_project_retrospective_revisions":    2,
				"workspace_project_retrospective_participants": 2,
				"workspace_project_retrospective_action_links": 1,
			} {
				assertProjectRetrospectiveProjectCount(t, db, table, `SELECT COUNT(*) FROM `+table+` WHERE workspace_id='workspace-1' AND project_id='project-1'`, want)
			}
			assertProjectRetrospectiveProjectCount(t, db, "resource revision", `SELECT COUNT(*) FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='project_retrospective' AND resource_id='retro-delete'`, 1)
			assertProjectRetrospectiveProjectCount(t, db, "project", `SELECT COUNT(*) FROM workspace_projects WHERE workspace_id='workspace-1' AND id='project-1'`, 1)
			assertProjectRetrospectiveProjectCount(t, db, "linked issue", `SELECT COUNT(*) FROM workspace_issues WHERE workspace_id='workspace-1' AND id='issue-retro-target' AND project_id='project-1'`, 1)
			assertProjectRetrospectiveProjectCount(t, db, "audit", `SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='project_retrospective' AND resource_id='retro-delete'`, 2)
			assertProjectRetrospectiveProjectCount(t, db, "outbox", `SELECT COUNT(*) FROM workspace_outbox_events WHERE aggregate_kind='project_retrospective' AND aggregate_id='retro-delete'`, 2)
		})
	}
}

func TestBothProjectDeletionRepositoriesFailClosedOnRetrospectiveOwnershipDrift(t *testing.T) {
	for _, useSurface := range []bool{true, false} {
		name := "installed local project repository"
		if useSurface {
			name = "canonical HTTP project surface repository"
		}
		t.Run(name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			seedProjectRetrospectiveDeletionAuthority(t, db)
			seedProjectRetrospectiveSameIDOwnershipDrift(t, db)
			if err := deleteProjectThroughInstalledRepositoryResult(db, useSurface); err == nil {
				t.Fatal("ownership-drift Project deletion error = nil")
			}
			for table, want := range map[string]int{
				"workspace_project_retrospectives":             1,
				"workspace_project_retrospective_revisions":    2,
				"workspace_project_retrospective_participants": 2,
				"workspace_project_retrospective_action_links": 1,
			} {
				assertProjectRetrospectiveProjectCount(t, db, table+" owning project", `SELECT COUNT(*) FROM `+table+` WHERE workspace_id='workspace-1' AND project_id='project-1'`, want)
			}
			for _, table := range []string{
				"workspace_project_retrospectives",
				"workspace_project_retrospective_revisions",
				"workspace_project_retrospective_participants",
				"workspace_project_retrospective_action_links",
			} {
				idColumn := "retrospective_id"
				if table == "workspace_project_retrospectives" {
					idColumn = "id"
				}
				assertProjectRetrospectiveProjectCount(t, db, table+" same-ID foreign project", `SELECT COUNT(*) FROM `+table+` WHERE workspace_id='workspace-1' AND project_id='project-2' AND `+idColumn+`='retro-delete'`, 1)
			}
			assertProjectRetrospectiveProjectCount(t, db, "resource revision", `SELECT COUNT(*) FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='project_retrospective' AND resource_id='retro-delete'`, 1)
			assertProjectRetrospectiveProjectCount(t, db, "project", `SELECT COUNT(*) FROM workspace_projects WHERE workspace_id='workspace-1' AND id='project-1'`, 1)
			assertProjectRetrospectiveProjectCount(t, db, "target", `SELECT COUNT(*) FROM workspace_issues WHERE workspace_id='workspace-1' AND id='issue-retro-target' AND project_id='project-1'`, 1)
		})
	}
}

func seedProjectRetrospectiveDeletionAuthority(t *testing.T, db *sql.DB) {
	t.Helper()
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 18, 0, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-delete",
		Content: retrospectiveDomain.Content{
			Summary: "Delete project safely", Lessons: []string{"Retain target"},
			ActionItems: []retrospectiveDomain.ActionItem{{ID: "action-1", Title: "Retained Issue"}},
		},
		IdempotencyKey: "retro-delete-create", RequestHash: strings.Repeat("5", 64), Actor: lead, OccurredAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, RequestID: "retro-delete-publish", Actor: lead, OccurredAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at)
			VALUES('issue-retro-target','workspace-1',91,'ONE-91','Retained target','todo','none','member','lead-1','project-1','2026-08-19T18:01:00Z','2026-08-19T18:01:00Z')`,
		`INSERT INTO workspace_project_retrospective_action_links(workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,target_id,request_hash,claimed_by,claimed_at,linked_by,linked_at)
			VALUES('workspace-1','project-1','retro-delete','action-1',2,'linked','issue','issue-retro-target','6666666666666666666666666666666666666666666666666666666666666666','lead-1','2026-08-19T18:02:00Z','lead-1','2026-08-19T18:02:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,lead_type,lead_id,asset_ids,created_at,updated_at)
			VALUES('project-2','workspace-1','Foreign project','in_progress','none','member','lead-member','[]','2026-08-19T18:00:00Z','2026-08-19T18:00:00Z')`,
		`INSERT INTO workspace_project_retrospectives(workspace_id,project_id,id,status,current_revision,created_by,created_at,updated_at)
			VALUES('workspace-1','project-2','retro-foreign','draft',1,'lead-1','2026-08-19T18:00:00Z','2026-08-19T18:00:00Z')`,
		`INSERT INTO workspace_project_retrospective_revisions(workspace_id,project_id,retrospective_id,revision,lifecycle_status,action,content_json,actor_id,created_at)
			VALUES('workspace-1','project-2','retro-foreign',1,'draft','create','{"summary":"Foreign","successes":[],"problems":[],"lessons":["Keep"],"action_items":[{"id":"foreign-action","title":"Keep"}]}','lead-1','2026-08-19T18:00:00Z')`,
		`INSERT INTO workspace_project_retrospective_participants(workspace_id,project_id,retrospective_id,revision,member_id,role)
			VALUES('workspace-1','project-2','retro-foreign',1,'lead-member','participant')`,
		`INSERT INTO workspace_project_retrospective_action_links(workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,request_hash,claimed_by,claimed_at)
			VALUES('workspace-1','project-2','retro-foreign','foreign-action',1,'pending','task','7777777777777777777777777777777777777777777777777777777777777777','lead-1','2026-08-19T18:00:00Z')`,
	} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func seedProjectRetrospectiveSameIDOwnershipDrift(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO workspace_project_retrospectives(workspace_id,project_id,id,status,current_revision,created_by,created_at,updated_at)
			VALUES('workspace-1','project-2','retro-delete','draft',1,'lead-1','2026-08-19T18:03:00Z','2026-08-19T18:03:00Z')`,
		`INSERT INTO workspace_project_retrospective_revisions(workspace_id,project_id,retrospective_id,revision,lifecycle_status,action,content_json,actor_id,created_at)
			VALUES('workspace-1','project-2','retro-delete',1,'draft','create','{"summary":"Drift","successes":[],"problems":[],"lessons":["Fail closed"],"action_items":[{"id":"drift-action","title":"Keep"}]}','lead-1','2026-08-19T18:03:00Z')`,
		`INSERT INTO workspace_project_retrospective_participants(workspace_id,project_id,retrospective_id,revision,member_id,role)
			VALUES('workspace-1','project-2','retro-delete',1,'lead-member','participant')`,
		`INSERT INTO workspace_project_retrospective_action_links(workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,request_hash,claimed_by,claimed_at)
			VALUES('workspace-1','project-2','retro-delete','drift-action',1,'pending','task','8888888888888888888888888888888888888888888888888888888888888888','lead-1','2026-08-19T18:03:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func deleteProjectThroughInstalledRepository(t *testing.T, db *sql.DB, useSurface bool) {
	t.Helper()
	if err := deleteProjectThroughInstalledRepositoryResult(db, useSurface); err != nil {
		t.Fatal(err)
	}
}

func deleteProjectThroughInstalledRepositoryResult(db *sql.DB, useSurface bool) error {
	now := time.Date(2026, 8, 19, 18, 5, 0, 0, time.UTC)
	if useSurface {
		repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
		if err != nil {
			return err
		}
		return repository.DeleteProject(context.Background(), "workspace-1", "project-1", now)
	}
	repository, err := persistence.NewProjectRepository(persistence.Config{DB: db})
	if err != nil {
		return err
	}
	return repository.DeleteWithDependents(context.Background(), "workspace-1", "project-1", now)
}

func assertProjectRetrospectiveProjectCount(t *testing.T, db *sql.DB, name, query string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil || count != want {
		t.Fatalf("%s count = %d, want %d, error %v", name, count, want, err)
	}
}
