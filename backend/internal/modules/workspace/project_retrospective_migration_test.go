package workspace

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

var projectRetrospectiveAuthorityTables = []string{
	"workspace_project_retrospectives",
	"workspace_project_retrospective_revisions",
	"workspace_project_retrospective_participants",
	"workspace_project_retrospective_action_links",
}

func TestProjectRetrospectiveMigrationInstallsAuthorityWithoutForbiddenDDL(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-retrospective-schema")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	assertProjectRetrospectiveTablesAndCatalog(t, db, len(projectRetrospectiveAuthorityTables), 1)

	contents, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000020_project_retrospectives.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := strings.ToUpper(string(contents))
	for _, forbidden := range []string{"FOREIGN KEY", "REFERENCES", "CASCADE", "TRIGGER", "CREATE INDEX", "CREATE UNIQUE INDEX"} {
		if strings.Contains(sqlText, forbidden) {
			t.Errorf("Project Retrospective migration contains forbidden DDL %q", forbidden)
		}
	}

	if _, err := db.Exec(`INSERT INTO workspace_project_retrospectives(
		workspace_id,project_id,id,status,current_revision,created_by,created_at,updated_at
	) VALUES
		('workspace-1','project-1','retro-1','draft',1,'member-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z'),
		('workspace-1','project-1','retro-1','draft',1,'member-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err == nil {
		t.Fatal("duplicate Retrospective identity error = nil")
	}
}

func TestProjectRetrospectiveEmptyDownRemovesOnlyNewAuthority(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-retrospective-down")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeProjectRetrospectiveDownForTest(context.Background(), db); err != nil {
		t.Fatalf("empty down error = %v", err)
	}
	assertProjectRetrospectiveTablesAndCatalog(t, db, 0, 0)
	var requirementCatalog int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000019_project_requirements.up.sql'`).Scan(&requirementCatalog); err != nil || requirementCatalog != 1 {
		t.Fatalf("predecessor catalog = %d, error %v", requirementCatalog, err)
	}
}

func TestProjectRetrospectiveMigrationUpgradesVersion19AndIsIdempotent(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-retrospective-upgrade")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeProjectRetrospectiveDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	assertProjectRetrospectiveTablesAndCatalog(t, db, 0, 0)
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatalf("version-19 upgrade error = %v", err)
	}
	assertProjectRetrospectiveTablesAndCatalog(t, db, len(projectRetrospectiveAuthorityTables), 1)
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatalf("idempotent second migrate error = %v", err)
	}
	assertProjectRetrospectiveTablesAndCatalog(t, db, len(projectRetrospectiveAuthorityTables), 1)
}

func TestProjectRetrospectiveDownBlocksEveryRetainedAuthorityAndPreservesState(t *testing.T) {
	cases := []struct {
		name      string
		statement string
		table     string
	}{
		{
			name: "head", table: "workspace_project_retrospectives",
			statement: `INSERT INTO workspace_project_retrospectives(workspace_id,project_id,id,status,current_revision,created_by,created_at,updated_at)
				VALUES('workspace-1','project-1','retro-1','draft',1,'member-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		},
		{
			name: "revision", table: "workspace_project_retrospective_revisions",
			statement: `INSERT INTO workspace_project_retrospective_revisions(workspace_id,project_id,retrospective_id,revision,lifecycle_status,action,content_json,actor_id,created_at)
				VALUES('workspace-1','project-1','retro-1',1,'draft','create','{"summary":"Summary","successes":[],"problems":[],"lessons":["Lesson"],"action_items":[]}','member-1','2026-08-19T00:00:00Z')`,
		},
		{
			name: "participant", table: "workspace_project_retrospective_participants",
			statement: `INSERT INTO workspace_project_retrospective_participants(workspace_id,project_id,retrospective_id,revision,member_id,role)
				VALUES('workspace-1','project-1','retro-1',1,'member-1','participant')`,
		},
		{
			name: "pending claim", table: "workspace_project_retrospective_action_links",
			statement: `INSERT INTO workspace_project_retrospective_action_links(workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,request_hash,claimed_by,claimed_at)
				VALUES('workspace-1','project-1','retro-1','action-1',2,'pending','task','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','member-1','2026-08-19T00:00:00Z')`,
		},
		{
			name: "completed link", table: "workspace_project_retrospective_action_links",
			statement: `INSERT INTO workspace_project_retrospective_action_links(workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,target_id,request_hash,claimed_by,claimed_at,linked_by,linked_at)
				VALUES('workspace-1','project-1','retro-1','action-1',2,'linked','issue','issue-1','bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','member-1','2026-08-19T00:00:00Z','member-1','2026-08-19T00:01:00Z')`,
		},
		{
			name: "resource revision", table: "workspace_resource_revisions",
			statement: `INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at)
				VALUES('workspace-1','project_retrospective','retro-1',1,'2026-08-19T00:00:00Z')`,
		},
		{
			name: "audit", table: "workspace_audit_entries",
			statement: `INSERT INTO workspace_audit_entries(workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json)
				VALUES('workspace-1','2026-08-19T00:00:00Z','audit-1','member','member-1','workspace.project.retrospective.create','project_retrospective','retro-1',1,'request-1','{}')`,
		},
		{
			name: "idempotency", table: "workspace_mutation_idempotency",
			statement: `INSERT INTO workspace_mutation_idempotency(workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at)
				VALUES('workspace-1','workspace.project.retrospective.create','key-1','cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc','project_retrospective','retro-1',1,201,'{}','2026-08-19T00:00:00Z')`,
		},
		{
			name: "outbox", table: "workspace_outbox_events",
			statement: `INSERT INTO workspace_outbox_events(state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,payload_json,actor_type,actor_id,attempt_count,created_at)
				VALUES('ready','2026-08-19T00:00:00Z','workspace-1','event-1','retrospective:drafted','project_retrospective','retro-1',1,'{}','member','member-1',0,'2026-08-19T00:00:00Z')`,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			db := openUnmigratedWorkspaceDB(t, "project-retrospective-retained-"+strings.ReplaceAll(testCase.name, " ", "-"))
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(testCase.statement); err != nil {
				t.Fatal(err)
			}
			beforeSchema := projectRetrospectiveSchemaSnapshot(t, db)
			var beforeRows int
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + testCase.table).Scan(&beforeRows); err != nil || beforeRows != 1 {
				t.Fatalf("before rows = %d, error %v", beforeRows, err)
			}
			if err := executeProjectRetrospectiveDownForTest(context.Background(), db); err == nil {
				t.Fatal("retained down error = nil")
			}
			var afterRows int
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + testCase.table).Scan(&afterRows); err != nil || afterRows != beforeRows {
				t.Fatalf("after rows = %d, want %d, error %v", afterRows, beforeRows, err)
			}
			if afterSchema := projectRetrospectiveSchemaSnapshot(t, db); afterSchema != beforeSchema {
				t.Fatalf("schema changed after blocked down\nbefore: %s\nafter: %s", beforeSchema, afterSchema)
			}
			assertProjectRetrospectiveTablesAndCatalog(t, db, len(projectRetrospectiveAuthorityTables), 1)
		})
	}
}

func executeProjectRetrospectiveDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000020_project_retrospectives.down.sql")
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

func assertProjectRetrospectiveTablesAndCatalog(t *testing.T, db *sql.DB, wantTables, wantCatalog int) {
	t.Helper()
	quoted := make([]string, len(projectRetrospectiveAuthorityTables))
	for index, table := range projectRetrospectiveAuthorityTables {
		quoted[index] = "'" + table + "'"
	}
	var tables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN (` + strings.Join(quoted, ",") + `)`).Scan(&tables); err != nil || tables != wantTables {
		t.Fatalf("Project Retrospective table count = %d, want %d, error %v", tables, wantTables, err)
	}
	var catalog int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000020_project_retrospectives.up.sql'`).Scan(&catalog); err != nil || catalog != wantCatalog {
		t.Fatalf("Project Retrospective catalog = %d, want %d, error %v", catalog, wantCatalog, err)
	}
}

func projectRetrospectiveSchemaSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT type,name,COALESCE(sql,'') FROM sqlite_master
		WHERE name IN ('workspace_project_retrospectives','workspace_project_retrospective_revisions','workspace_project_retrospective_participants','workspace_project_retrospective_action_links','workspace_schema_migrations')
		ORDER BY type,name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	parts := make([]string, 0, 5)
	for rows.Next() {
		var objectType, name, sqlText string
		if err = rows.Scan(&objectType, &name, &sqlText); err != nil {
			t.Fatal(err)
		}
		parts = append(parts, objectType+":"+name+":"+sqlText)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(parts, "\n")
}
