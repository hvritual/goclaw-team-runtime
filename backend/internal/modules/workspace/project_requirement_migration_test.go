package workspace

import (
	"context"
	"database/sql"
	"io/fs"
	"strings"
	"testing"
)

var projectRequirementAuthorityTables = []string{
	"workspace_requirement_baselines",
	"workspace_requirement_revisions",
	"workspace_requirement_issue_links",
	"workspace_requirement_outline_links",
	"workspace_requirement_review_projections",
	"workspace_project_requirement_access_sets",
	"workspace_project_requirement_grants",
	"workspace_project_outline_sets",
	"workspace_project_outline_nodes",
}

func TestProjectRequirementMigrationInstallsSingularAuthorityWithoutExplicitIndexes(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-requirement-schema")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	assertProjectRequirementTablesAndCatalog(t, db, len(projectRequirementAuthorityTables), 1)

	contents, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000019_project_requirements.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := strings.ToUpper(string(contents))
	for _, forbidden := range []string{"FOREIGN KEY", "REFERENCES", "CASCADE", "TRIGGER", "CREATE INDEX", "CREATE UNIQUE INDEX"} {
		if strings.Contains(sqlText, forbidden) {
			t.Errorf("Project Requirement migration contains forbidden DDL %q", forbidden)
		}
	}

	if _, err := db.Exec(`INSERT INTO workspace_requirement_baselines(
		id,workspace_id,project_id,status,current_revision,latest_content_author,created_at,updated_at
	) VALUES
		('baseline-1','workspace-1','project-1','draft',1,'member-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z'),
		('baseline-2','workspace-1','project-1','draft',1,'member-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err == nil {
		t.Fatal("duplicate project Requirement baseline error = nil")
	}
}

func TestProjectRequirementMigrationImportsValidLegacyBaselineAndReversesUntouchedImport(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-requirement-legacy-import")
	applyWorkspaceMigrationsBeforeProjectRequirements(t, db)
	insertLegacyRequirementFixture(t, db, "requirement-1", "project-1", `["ACM-1"]`)
	if _, err := db.Exec(`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES
		('version-2','requirement-1',2,'Second legacy body','2026-08-19T00:01:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE workspace_requirements SET current_version=2,updated_at='2026-08-19T00:01:00Z' WHERE id='requirement-1'`); err != nil {
		t.Fatal(err)
	}

	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var status, legacyID, author string
	var revision int64
	if err := db.QueryRow(`SELECT status,current_revision,legacy_requirement_id,latest_content_author
		FROM workspace_requirement_baselines WHERE id='requirement-1'`).Scan(&status, &revision, &legacyID, &author); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || revision != 2 || legacyID != "requirement-1" || author != "legacy-import" {
		t.Fatalf("imported baseline = status %q revision %d legacy %q author %q", status, revision, legacyID, author)
	}
	var problem, goalKey, goalText string
	if err := db.QueryRow(`SELECT
		json_extract(content_json,'$.problem_statement'),
		json_extract(content_json,'$.goals[0].key'),
		json_extract(content_json,'$.goals[0].text')
		FROM workspace_requirement_revisions WHERE baseline_id='requirement-1' AND revision=2`).Scan(&problem, &goalKey, &goalText); err != nil {
		t.Fatal(err)
	}
	if problem != "Second legacy body" || goalKey != "legacy-root" || goalText != "Legacy title" {
		t.Fatalf("imported content = problem %q key %q text %q", problem, goalKey, goalText)
	}
	var linkedIssue string
	if err := db.QueryRow(`SELECT issue_id FROM workspace_requirement_issue_links WHERE baseline_id='requirement-1'`).Scan(&linkedIssue); err != nil {
		t.Fatal(err)
	}
	if linkedIssue != "issue-1" {
		t.Fatalf("imported Issue link = %q", linkedIssue)
	}
	var legacyRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_requirements WHERE id='requirement-1'`).Scan(&legacyRows); err != nil || legacyRows != 1 {
		t.Fatalf("legacy rows = %d, %v", legacyRows, err)
	}

	if err := executeProjectRequirementDownForTest(context.Background(), db); err != nil {
		t.Fatalf("untouched legacy import down error = %v", err)
	}
	assertProjectRequirementTablesAndCatalog(t, db, 0, 0)
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_requirements WHERE id='requirement-1'`).Scan(&legacyRows); err != nil || legacyRows != 1 {
		t.Fatalf("retained legacy rows after down = %d, %v", legacyRows, err)
	}
}

func TestProjectRequirementMigrationRejectsAmbiguousLegacyProjectAuthority(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-requirement-legacy-duplicate")
	applyWorkspaceMigrationsBeforeProjectRequirements(t, db)
	insertLegacyRequirementFixture(t, db, "requirement-1", "project-1", `[]`)
	if _, err := db.Exec(`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at)
		VALUES('requirement-2','workspace-1','project-1','Second',1,'draft','uncovered','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at)
		VALUES('version-2','requirement-2',1,'Second body','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err == nil {
		t.Fatal("MigrateSqlite() error = nil for duplicate legacy project baselines")
	}
	assertProjectRequirementTablesAndCatalog(t, db, 0, 0)
	var legacyRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_requirements`).Scan(&legacyRows); err != nil || legacyRows != 2 {
		t.Fatalf("legacy rows after rollback = %d, %v", legacyRows, err)
	}
}

func TestProjectRequirementMigrationRejectsForeignLegacyIssueReference(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-requirement-legacy-foreign-issue")
	applyWorkspaceMigrationsBeforeProjectRequirements(t, db)
	insertLegacyRequirementFixture(t, db, "requirement-1", "project-1", `["issue-2"]`)
	if _, err := db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,status,created_at,updated_at)
		VALUES('project-2','workspace-1','Second','planned','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at)
		VALUES('issue-2','workspace-1',2,'ACM-2','Foreign','todo','none','member','member-1','project-2','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSqlite(context.Background(), db); err == nil {
		t.Fatal("MigrateSqlite() error = nil for foreign legacy Issue reference")
	}
	assertProjectRequirementTablesAndCatalog(t, db, 0, 0)
}

func TestProjectRequirementDownMigrationRemovesEmptyAuthority(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-requirement-down-empty")
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := executeProjectRequirementDownForTest(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	assertProjectRequirementTablesAndCatalog(t, db, 0, 0)
}

func TestProjectRequirementDownMigrationRejectsLegacySnapshotDriftWithoutChangingRows(t *testing.T) {
	db := openUnmigratedWorkspaceDB(t, "project-requirement-down-legacy-drift")
	applyWorkspaceMigrationsBeforeProjectRequirements(t, db)
	insertLegacyRequirementFixture(t, db, "requirement-1", "project-1", `["ACM-1"]`)
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE workspace_requirements SET issue_ids='["issue-1"]' WHERE id='requirement-1'`); err != nil {
		t.Fatal(err)
	}
	var beforeLegacy, beforeCanonical string
	if err := db.QueryRow(`SELECT json_array(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at)
		FROM workspace_requirements WHERE id='requirement-1'`).Scan(&beforeLegacy); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT json_array(id,workspace_id,project_id,status,current_revision,legacy_requirement_id)
		FROM workspace_requirement_baselines WHERE id='requirement-1'`).Scan(&beforeCanonical); err != nil {
		t.Fatal(err)
	}
	if err := executeProjectRequirementDownForTest(context.Background(), db); err == nil {
		t.Fatal("Project Requirement down succeeded after legacy snapshot drift")
	}
	var afterLegacy, afterCanonical string
	if err := db.QueryRow(`SELECT json_array(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at)
		FROM workspace_requirements WHERE id='requirement-1'`).Scan(&afterLegacy); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT json_array(id,workspace_id,project_id,status,current_revision,legacy_requirement_id)
		FROM workspace_requirement_baselines WHERE id='requirement-1'`).Scan(&afterCanonical); err != nil {
		t.Fatal(err)
	}
	if beforeLegacy != afterLegacy || beforeCanonical != afterCanonical {
		t.Fatalf("blocked down changed rows\nlegacy before: %s\n legacy after: %s\ncanonical before: %s\n canonical after: %s", beforeLegacy, afterLegacy, beforeCanonical, afterCanonical)
	}
	assertProjectRequirementTablesAndCatalog(t, db, len(projectRequirementAuthorityTables), 1)
}

func TestProjectRequirementDownMigrationRejectsImportedIssueLinkOwnershipDriftWithoutChangingRows(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		mutation string
	}{
		{name: "workspace ownership", mutation: `UPDATE workspace_requirement_issue_links
			SET workspace_id='other-workspace' WHERE baseline_id='requirement-1' AND issue_id='issue-1'`},
		{name: "project ownership", mutation: `UPDATE workspace_requirement_issue_links
			SET project_id='other-project' WHERE baseline_id='requirement-1' AND issue_id='issue-1'`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := openUnmigratedWorkspaceDB(t, "project-requirement-down-link-ownership")
			applyWorkspaceMigrationsBeforeProjectRequirements(t, db)
			insertLegacyRequirementFixture(t, db, "requirement-1", "project-1", `["ACM-1"]`)
			if err := MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(testCase.mutation); err != nil {
				t.Fatal(err)
			}
			before := projectRequirementImportedIssueLinkSnapshot(t, db)

			if err := executeProjectRequirementDownForTest(context.Background(), db); err == nil {
				t.Fatal("Project Requirement down succeeded after imported Issue-link ownership drift")
			}

			after := projectRequirementImportedIssueLinkSnapshot(t, db)
			if before != after {
				t.Fatalf("blocked down changed imported rows\nbefore: %s\n after: %s", before, after)
			}
			assertProjectRequirementTablesAndCatalog(t, db, len(projectRequirementAuthorityTables), 1)
		})
	}
}

func projectRequirementImportedIssueLinkSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	queries := []string{
		`SELECT json_array(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at)
			FROM workspace_requirements WHERE id='requirement-1'`,
		`SELECT json_array(id,workspace_id,project_id,status,current_revision,legacy_requirement_id,legacy_snapshot_json,created_at,updated_at)
			FROM workspace_requirement_baselines WHERE id='requirement-1'`,
		`SELECT json_array(baseline_id,revision,content_json,status,action,change_summary,actor_id,created_at)
			FROM workspace_requirement_revisions WHERE baseline_id='requirement-1' ORDER BY revision`,
		`SELECT json_array(baseline_id,workspace_id,project_id,requirement_key,issue_id,linked_revision,unlinked_revision,linked_by,linked_at,unlinked_by,unlinked_at)
			FROM workspace_requirement_issue_links WHERE baseline_id='requirement-1' AND issue_id='issue-1'`,
	}
	rows := make([]string, 0, len(queries))
	for _, query := range queries {
		var row string
		if err := db.QueryRow(query).Scan(&row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}

func insertLegacyRequirementFixture(t *testing.T, db *sql.DB, requirementID, projectID, issueIDs string) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES('workspace-1','Acme','acme','ACM','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,created_at,updated_at) VALUES('project-1','workspace-1','Project','planned','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at) VALUES('issue-1','workspace-1',1,'ACM-1','Linked','todo','none','member','member-1','project-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at)
		VALUES(?,'workspace-1',?,'Legacy title',1,'draft','covered',?,'2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`, requirementID, projectID, issueIDs); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at)
		VALUES('version-1',?,1,'Legacy body','2026-08-19T00:00:00Z')`, requirementID); err != nil {
		t.Fatal(err)
	}
}

func applyWorkspaceMigrationsBeforeProjectRequirements(t *testing.T, db *sql.DB) {
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
		if version >= "000019_" {
			continue
		}
		migration, readErr := sqliteMigrationFiles.ReadFile(migrationPath)
		if readErr != nil {
			t.Fatal(readErr)
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

func executeProjectRequirementDownForTest(ctx context.Context, db *sql.DB) error {
	down, err := sqliteMigrationFiles.ReadFile(SqliteMigrationDir() + "/000019_project_requirements.down.sql")
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

func assertProjectRequirementTablesAndCatalog(t *testing.T, db *sql.DB, wantTables, wantCatalog int) {
	t.Helper()
	quoted := make([]string, len(projectRequirementAuthorityTables))
	for index, table := range projectRequirementAuthorityTables {
		quoted[index] = "'" + table + "'"
	}
	var tables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN (` + strings.Join(quoted, ",") + `)`).Scan(&tables); err != nil || tables != wantTables {
		t.Fatalf("Project Requirement table count = %d, want %d, error %v", tables, wantTables, err)
	}
	var catalog int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_schema_migrations WHERE version='000019_project_requirements.up.sql'`).Scan(&catalog); err != nil || catalog != wantCatalog {
		t.Fatalf("Project Requirement catalog = %d, want %d, error %v", catalog, wantCatalog, err)
	}
}
