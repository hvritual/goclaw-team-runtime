package workspace

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	_ "modernc.org/sqlite"
)

func TestWorkspaceGovernanceMigrationUpgradesRetainedVersionEightDatabase(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "governance-upgrade")
	applyWorkspaceMigrationsBeforeGovernance(t, db)
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES('workspace-1','Acme','acme','ACM','2026-08-16T00:00:00Z','2026-08-16T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations`).Scan(&migrationCount); err != nil || migrationCount != 18 {
		t.Fatalf("migration count = %d, %v", migrationCount, err)
	}
	var workspaceName string
	if err := db.QueryRow(`SELECT name FROM workspaces WHERE id='workspace-1'`).Scan(&workspaceName); err != nil || workspaceName != "Acme" {
		t.Fatalf("retained workspace = %q, %v", workspaceName, err)
	}
}

func TestPinReorderMigrationBackfillsAndGuardsAdvancedRevision(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "pin-reorder")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executePinReorderDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('pin-1','workspace-1','user-1','issue','issue-1',1,'2026-08-18T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var revision int64
	if err := db.QueryRow(`SELECT revision FROM workspace_pin_order_revisions WHERE workspace_id='workspace-1' AND user_id='user-1'`).Scan(&revision); err != nil || revision != 1 {
		t.Fatalf("backfilled revision = %d, %v", revision, err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_pins(id,workspace_id,user_id,item_type,item_id,position,created_at) VALUES('pin-2','workspace-1','user-1','project','project-1',2,'2026-08-18T00:00:01Z')`); err != nil {
		t.Fatal(err)
	}
	if err := executePinReorderDownForTest(context.Background(), db); err == nil {
		t.Fatal("pin reorder down succeeded with advanced revision")
	}
	var tableCount, catalogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='workspace_pin_order_revisions'`).Scan(&tableCount); err != nil || tableCount != 1 {
		t.Fatalf("revision table count = %d, %v", tableCount, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000015_pin_reorder.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 1 {
		t.Fatalf("revision catalog count = %d, %v", catalogCount, err)
	}
}

func TestKnowledgeQueryDownMigrationRejectsEveryRetainedTable(t *testing.T) {
	for _, test := range []struct {
		name      string
		statement string
	}{
		{
			name:      "governed entry",
			statement: `INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-1','workspace-1','procedure','published',1,'2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		},
		{
			name:      "orphan revision",
			statement: `INSERT INTO workspace_knowledge_revisions(knowledge_id,revision,supersedes_revision,title,content,created_by,created_at) VALUES('knowledge-1',1,0,'Title','Body','user-1','2026-08-18T00:00:00Z')`,
		},
		{
			name:      "orphan source reference",
			statement: `INSERT INTO workspace_knowledge_source_refs(knowledge_id,revision,ordinal,source_type,source_id,source_revision,citation) VALUES('knowledge-1',1,0,'acceptance_conclusion','issue-1','sha256:abc','Accepted')`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openUnmigratedWorkspaceDB(t, "knowledge-query-down-"+strings.ReplaceAll(test.name, " ", "-"))
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(test.statement); err != nil {
				t.Fatal(err)
			}
			if err := executeKnowledgeQueryDownForTest(context.Background(), db); err == nil {
				t.Fatal("knowledge query down succeeded with retained data")
			}
			var tableCount, catalogCount int
			if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('workspace_governed_knowledge','workspace_knowledge_revisions','workspace_knowledge_source_refs')`).Scan(&tableCount); err != nil || tableCount != 3 {
				t.Fatalf("knowledge table count = %d, %v", tableCount, err)
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000016_knowledge_query.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 1 {
				t.Fatalf("knowledge catalog count = %d, %v", catalogCount, err)
			}
		})
	}
}

func TestKnowledgeReviewDownMigrationRejectsEveryRetainedDependency(t *testing.T) {
	for _, test := range []struct{ name, statement string }{
		{"candidate", `INSERT INTO workspace_knowledge_candidates(id,workspace_id,target_revision,kind,title,content,reason,status,revision,proposed_by,created_at,updated_at) VALUES('candidate-1','workspace-1',0,'lesson','Title','Body','Reason','candidate',1,'user-1','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`},
		{"candidate source", `INSERT INTO workspace_knowledge_candidate_sources(candidate_id,ordinal,source_type,source_id,source_revision,citation) VALUES('candidate-1',0,'acceptance_conclusion','issue-1','sha256:abc','Accepted')`},
		{"review event", `INSERT INTO workspace_knowledge_review_events(candidate_id,candidate_revision,action,actor_id,rationale,emergency,occurred_at) VALUES('candidate-1',2,'approve','user-2','Approved',0,'2026-08-18T00:00:00Z')`},
		{"publication", `INSERT INTO workspace_knowledge_publications(candidate_id,knowledge_id,action,created_at) VALUES('candidate-1','knowledge-1','publish','2026-08-18T00:00:00Z')`},
		{"invalidated entry", `INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-1','workspace-1','lesson','invalidated',1,'2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`},
		{"audit", `INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json) VALUES('workspace-1','2026-08-18T00:00:00Z','audit-1','member','user-1','workspace.knowledge.propose','knowledge_candidate','candidate-1',1,'request-1','{}')`},
		{"idempotency", `INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES('workspace-1','workspace.knowledge.propose','key-1','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','knowledge_candidate','candidate-1',1,201,'{}','2026-08-18T00:00:00Z')`},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openUnmigratedWorkspaceDB(t, "knowledge-review-down-"+strings.ReplaceAll(test.name, " ", "-"))
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(test.statement); err != nil {
				t.Fatal(err)
			}
			if err := executeKnowledgeReviewDownForTest(context.Background(), db); err == nil {
				t.Fatal("knowledge review down succeeded with retained dependency")
			}
			var catalog int
			if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000017_knowledge_review.up.sql'`).Scan(&catalog); err != nil || catalog != 1 {
				t.Fatalf("review catalog = %d, %v", catalog, err)
			}
		})
	}
}

func TestProjectResourceDownMigrationRejectsEveryRetainedDependency(t *testing.T) {
	for _, test := range []struct{ name, statement string }{
		{
			"resource",
			`INSERT INTO workspace_project_resources(
				id,workspace_id,project_id,resource_type,canonical_url,resource_ref,fingerprint,label,position,status,revision,
				connection_state,connection_diagnostic_code,created_at,created_by,updated_at,updated_by
			) VALUES('resource-1','workspace-1','project-1','url','https://example.com/docs','','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','Docs',0,'active',1,'unchecked','','2026-08-19T00:00:00Z','user-1','2026-08-19T00:00:00Z','user-1')`,
		},
		{
			"revision zero",
			`INSERT INTO workspace_project_resource_sets(workspace_id,project_id,revision,updated_at) VALUES('workspace-1','project-1',0,'2026-08-19T00:00:00Z')`,
		},
		{
			"audit resource kind",
			`INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json) VALUES('workspace-1','2026-08-19T00:00:00Z','audit-1','member','user-1','workspace.unrelated.action','project_resource','resource-1',1,'request-1','{}')`,
		},
		{
			"audit action namespace",
			`INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json) VALUES('workspace-1','2026-08-19T00:00:00Z','audit-1','member','user-1','workspace.project.resource.restore','unrelated','resource-1',1,'request-1','{}')`,
		},
		{
			"idempotency resource kind",
			`INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES('workspace-1','workspace.unrelated.action','key-1','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','project_resource','resource-1',1,201,'{}','2026-08-19T00:00:00Z')`,
		},
		{
			"idempotency action namespace",
			`INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES('workspace-1','workspace.project.resource.restore','key-1','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','unrelated','resource-1',1,200,'{}','2026-08-19T00:00:00Z')`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openUnmigratedWorkspaceDB(t, "project-resource-down-"+test.name)
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(test.statement); err != nil {
				t.Fatal(err)
			}
			if err := executeProjectResourceDownForTest(context.Background(), db); err == nil {
				t.Fatal("Project Resource down succeeded with retained dependency")
			}
			var tables, catalog int
			if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('workspace_project_resources','workspace_project_resource_sets')`).Scan(&tables); err != nil || tables != 2 {
				t.Fatalf("Project Resource table count = %d, %v", tables, err)
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000018_project_resources.up.sql'`).Scan(&catalog); err != nil || catalog != 1 {
				t.Fatalf("Project Resource catalog = %d, %v", catalog, err)
			}
		})
	}
}

func TestProjectResourceMigrationUsesNoExplicitIndexes(t *testing.T) {
	contents, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000018_project_resources.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := strings.ToUpper(string(contents))
	for _, forbidden := range []string{"FOREIGN KEY", "REFERENCES", "CASCADE", "TRIGGER", "CREATE INDEX", "CREATE UNIQUE INDEX"} {
		if strings.Contains(sqlText, forbidden) {
			t.Errorf("Project Resource migration contains forbidden DDL %q", forbidden)
		}
	}
}

func TestProjectResourceDownMigrationRemovesEmptyAuthority(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-resource-down-empty")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeProjectResourceDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var tables, catalog int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('workspace_project_resources','workspace_project_resource_sets')`).Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("Project Resource table count = %d, %v", tables, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000018_project_resources.up.sql'`).Scan(&catalog); err != nil || catalog != 0 {
		t.Fatalf("Project Resource catalog = %d, %v", catalog, err)
	}
}

func TestTaskIssuePromotionMigrationInstallsImmutableLink(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "task-issue-promotion")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES('workspace-1','Acme','acme','ACM','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_todos(id,workspace_id,title,status,creator_type,creator_id,created_at,updated_at) VALUES('task-1','workspace-1','Promote me','todo','member','member-1','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,created_at,updated_at) VALUES('issue-1','workspace-1',1,'ACM-1','Promoted','todo','none','member','member-1','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`,
		`INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at,response_snapshot) VALUES('workspace-1','task-1','issue-1','2026-08-18T00:00:00Z','{}')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at,response_snapshot) VALUES('workspace-1','task-1','issue-2','2026-08-18T00:00:00Z','{}')`); err == nil {
		t.Fatal("duplicate source Task error = nil")
	}
	if _, err := db.Exec(`INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at,response_snapshot) VALUES('workspace-1','task-2','issue-1','2026-08-18T00:00:00Z','{}')`); err == nil {
		t.Fatal("duplicate promoted Issue error = nil")
	}
	if _, err := db.Exec(`INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at) VALUES('workspace-1','task-3','issue-3','2026-08-18T00:00:00Z')`); err == nil {
		t.Fatal("missing response snapshot error = nil")
	}
	if _, err := db.Exec(`INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at,response_snapshot) VALUES('workspace-1','task-4','issue-4','2026-08-18T00:00:00Z','not-json')`); err == nil {
		t.Fatal("invalid response snapshot JSON error = nil")
	}
}

func TestWorkspaceGovernanceMigrationRollsBackPartialVersionNineFailure(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "governance-rollback")
	applyWorkspaceMigrationsBeforeGovernance(t, db)
	if _, err := db.Exec(`CREATE VIEW workspace_audit_entries AS SELECT 'blocked' AS id`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err == nil {
		t.Fatal("MigrateSqlite() error = nil")
	}
	for _, table := range []string{"workspace_resource_revisions", "workspace_mutation_idempotency", "workspace_outbox_events"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rolled-back table %s count = %d, %v", table, count, err)
		}
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations`).Scan(&migrationCount); err != nil || migrationCount != 8 {
		t.Fatalf("migration count after rollback = %d, %v", migrationCount, err)
	}
}

func TestWorkspaceGovernanceRowsPersistAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "governance-restart.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES('workspace-1','task','task-1',3,'2026-08-16T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	if err := MigrateSqlite(context.Background(), restarted); err != nil {
		t.Fatal(err)
	}
	var revision int64
	if err := restarted.QueryRow(`SELECT revision FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='task' AND resource_id='task-1'`).Scan(&revision); err != nil || revision != 3 {
		t.Fatalf("retained revision = %d, %v", revision, err)
	}
}

func TestTaskCursorSigningKeyIsSecretAndPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task-cursor-key.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	first, err := loadOrCreateTaskCursorSigningKey(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 {
		t.Fatalf("signing key length = %d", len(first))
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	restarted, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	if err := MigrateSqlite(context.Background(), restarted); err != nil {
		t.Fatal(err)
	}
	second, err := loadOrCreateTaskCursorSigningKey(context.Background(), restarted)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != string(first) {
		t.Fatal("task cursor signing key changed across restart")
	}
}

func TestTaskCursorSigningDownRemovesCatalogAndCanReapply(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "task-cursor-signing-down")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateTaskCursorSigningKey(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeTaskCursorSigningDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var catalogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000011_task_cursor_signing.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 0 {
		t.Fatalf("task cursor signing catalog count = %d, %v", catalogCount, err)
	}
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateTaskCursorSigningKey(context.Background(), db); err != nil {
		t.Fatalf("load key after reapply: %v", err)
	}
}

func TestWorkspaceGovernanceDownRejectsEveryNonEmptyGovernanceTableAtomically(t *testing.T) {
	cases := map[string]string{
		"resource revisions": `INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at) VALUES('workspace-1','task','task-1',1,'2026-08-17T00:00:00Z')`,
		"idempotency":        `INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at) VALUES('workspace-1','workspace.task.create','key-1','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','task','task-1',1,201,'{"version":"governance-replay-v1","data":{"id":"task-1"}}','2026-08-17T00:00:00Z')`,
		"audit":              `INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json) VALUES('workspace-1','2026-08-17T00:00:00Z','audit-1','member','user-1','workspace.task.create','task','task-1',1,'request-1','{"version":"governance-audit-v1","data":{"status":"todo"}}')`,
		"outbox":             `INSERT INTO workspace_outbox_events(state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,payload_json,actor_type,actor_id,created_at) VALUES('ready','2026-08-17T00:00:00Z','workspace-1','event-1','task:created','task','task-1',1,'{"version":"governance-outbox-v1","data":{"id":"task-1"}}','member','user-1','2026-08-17T00:00:00Z')`,
	}
	for name, insert := range cases {
		t.Run(name, func(t *testing.T) {
			db := openUnmigratedWorkspaceDB(t, "governance-down-"+strings.ReplaceAll(name, " ", "-"))
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(insert); err != nil {
				t.Fatal(err)
			}
			if err := executeWorkspaceGovernanceDownForTest(context.Background(), db); err == nil {
				t.Fatal("governance down succeeded with retained evidence")
			}
			assertGovernanceTablesAndCatalog(t, db, 4, 1)
		})
	}
}

func TestWorkspaceGovernanceDownRemovesOnlyEmptyTablesAndCatalogEntry(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "governance-down-empty")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeWorkspaceGovernanceDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	assertGovernanceTablesAndCatalog(t, db, 0, 0)
}

func TestTaskLifecycleDownRemovesEmptyLifecycleColumnsAndCatalogEntry(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "task-lifecycle-down-empty")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeTaskLifecycleDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"revision", "restore_status", "archived_at"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('workspace_todos') WHERE name=?`, column).Scan(&count); err != nil || count != 0 {
			t.Fatalf("column %s count = %d, %v", column, count, err)
		}
	}
	var catalogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000010_task_lifecycle.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 0 {
		t.Fatalf("task lifecycle catalog count = %d, %v", catalogCount, err)
	}
}

func TestTaskLifecycleDownRejectsArchivedTaskAtomically(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "task-lifecycle-down-archived")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_todos(
		id,workspace_id,title,status,created_at,updated_at,priority,creator_type,creator_id,
		revision,restore_status,archived_at
	) VALUES('task-1','workspace-1','Archived','archived','2026-08-18T00:00:00Z',
		'2026-08-18T00:00:00Z','none','member','member-1',2,'cancelled','2026-08-18T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := executeTaskLifecycleDownForTest(context.Background(), db); err == nil {
		t.Fatal("task lifecycle down succeeded with archived data")
	}
	var status string
	var revision int64
	if err := db.QueryRow(`SELECT status,revision FROM workspace_todos WHERE id='task-1'`).Scan(&status, &revision); err != nil || status != "archived" || revision != 2 {
		t.Fatalf("retained task = %s/%d, %v", status, revision, err)
	}
	var catalogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000010_task_lifecycle.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 1 {
		t.Fatalf("retained task lifecycle catalog count = %d, %v", catalogCount, err)
	}
}

func TestTaskIssuePromotionDownRequiresEmptyTable(t *testing.T) {
	t.Run("empty table rolls back", func(t *testing.T) {
		db := openUnmigratedWorkspaceDB(t, "task-promotion-down-empty")
		if err := MigrateSqlite(context.Background(), db); err != nil {
			t.Fatal(err)
		}
		if err := executeTaskIssuePromotionDownForTest(context.Background(), db); err != nil {
			t.Fatal(err)
		}
		var tableCount, catalogCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='workspace_task_issue_promotions'`).Scan(&tableCount); err != nil || tableCount != 0 {
			t.Fatalf("promotion table count = %d, %v", tableCount, err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000012_task_issue_promotion.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 0 {
			t.Fatalf("promotion catalog count = %d, %v", catalogCount, err)
		}
	})

	t.Run("populated table is retained", func(t *testing.T) {
		db := openUnmigratedWorkspaceDB(t, "task-promotion-down-populated")
		if err := MigrateSqlite(context.Background(), db); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO workspace_task_issue_promotions(workspace_id,task_id,issue_id,created_at,response_snapshot) VALUES('workspace-1','task-1','issue-1','2026-08-18T00:00:00Z','{}')`); err != nil {
			t.Fatal(err)
		}
		if err := executeTaskIssuePromotionDownForTest(context.Background(), db); err == nil {
			t.Fatal("promotion down succeeded with retained link")
		}
		var linkCount, catalogCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_task_issue_promotions`).Scan(&linkCount); err != nil || linkCount != 1 {
			t.Fatalf("retained promotion links = %d, %v", linkCount, err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000012_task_issue_promotion.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != 1 {
			t.Fatalf("retained promotion catalog count = %d, %v", catalogCount, err)
		}
	})
}

func executeTaskIssuePromotionDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000012_task_issue_promotion.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executePinReorderDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000015_pin_reorder.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executeKnowledgeQueryDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000016_knowledge_query.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executeKnowledgeReviewDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000017_knowledge_review.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executeProjectResourceDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000018_project_resources.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executeTaskLifecycleDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000010_task_lifecycle.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executeTaskCursorSigningDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000011_task_cursor_signing.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func executeWorkspaceGovernanceDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000009_workspace_governance.down.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(down)); err != nil {
		return err
	}
	return tx.Commit()
}

func assertGovernanceTablesAndCatalog(t *testing.T, db *sql.DB, wantTables, wantCatalog int) {
	t.Helper()
	var tableCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN (
		'workspace_resource_revisions','workspace_mutation_idempotency','workspace_audit_entries','workspace_outbox_events'
	)`).Scan(&tableCount); err != nil || tableCount != wantTables {
		t.Fatalf("governance table count = %d, want %d, error %v", tableCount, wantTables, err)
	}
	var catalogCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000009_workspace_governance.up.sql'`).Scan(&catalogCount); err != nil || catalogCount != wantCatalog {
		t.Fatalf("governance catalog count = %d, want %d, error %v", catalogCount, wantCatalog, err)
	}
}

func openUnmigratedWorkspaceDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func applyWorkspaceMigrationsBeforeGovernance(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`CREATE TABLE workspace_schema_migrations(version TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	paths, err := fs.Glob(sqliteMigrationFiles, SqliteMigrationDir()+"/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, migrationPath := range paths {
		version := migrationPath[len(SqliteMigrationDir())+1:]
		if version >= "000009_" {
			continue
		}
		migration, err := sqliteMigrationFiles.ReadFile(migrationPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(string(migration)); err != nil {
			t.Fatalf("apply %s: %v", migrationPath, err)
		}
		if _, err := tx.Exec(`INSERT INTO workspace_schema_migrations(version) VALUES(?)`, version); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestSqliteWorkspaceIdentityReaderFindsOnlyRequestedWorkspace(t *testing.T) {
	db, err := sql.Open("sqlite", "file:workspace-identity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := MigrateSqlite(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, workspace := range []struct {
		id   string
		name string
		slug string
	}{
		{id: "workspace-1", name: "Acme", slug: "acme"},
		{id: "workspace-2", name: "Globex", slug: "globex"},
	} {
		_, err = db.ExecContext(ctx, `INSERT INTO workspaces(
			id, name, slug, issue_prefix, created_at, updated_at
		) VALUES (?, ?, ?, 'WSP', '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`,
			workspace.id, workspace.name, workspace.slug)
		if err != nil {
			t.Fatal(err)
		}
	}

	module, err := NewWithSqlitePersistence(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	message, err := module.Local().Ping(ctx)
	if err != nil || message != "pong" {
		t.Fatalf("persistent module Ping() = %q, %v", message, err)
	}
	identity, err := module.IdentityLocal().FindIdentity(ctx, "workspace-1")
	if err != nil {
		t.Fatal(err)
	}
	if identity != (contract.WorkspaceIdentity{ID: "workspace-1", Name: "Acme"}) {
		t.Fatalf("unexpected workspace identity: %+v", identity)
	}
	if _, err := module.IdentityLocal().FindIdentity(ctx, "missing"); !errors.Is(err, contract.ErrWorkspaceNotFound) {
		t.Fatalf("missing workspace error = %v", err)
	}
}

func TestWorkspaceGovernanceMigrationUsesOnlyCompositePrimaryKeyAccessPaths(t *testing.T) {
	contents, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000009_workspace_governance.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := strings.ToUpper(string(contents))
	if got := strings.Count(sqlText, "WITHOUT ROWID"); got != 4 {
		t.Fatalf("WITHOUT ROWID count = %d, want 4", got)
	}
	for _, forbidden := range []string{
		"FOREIGN KEY", "REFERENCES", "CASCADE", "TRIGGER",
		"CREATE INDEX", "CREATE UNIQUE INDEX",
	} {
		if strings.Contains(sqlText, forbidden) {
			t.Errorf("governance migration contains forbidden DDL %q", forbidden)
		}
	}
}

func TestNewWithSqlitePersistenceRejectsMissingWorkspaceDatabase(t *testing.T) {
	if _, err := NewWithSqlitePersistence(SqlitePersistenceConfig{}); err == nil {
		t.Fatal("NewWithSqlitePersistence() error = nil")
	}
}
