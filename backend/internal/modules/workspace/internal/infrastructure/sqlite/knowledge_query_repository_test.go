package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	workspace "github.com/hvritual/workspace/internal/modules/workspace"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestKnowledgeQueryRepositoryLoadsImmutableSourcesWithinWorkspace(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-query.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-1','workspace-1','lesson','published',2,'2026-08-18T01:00:00Z','2026-08-18T02:00:00Z')`,
		`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('foreign','workspace-2','lesson','published',1,'2026-08-18T01:00:00Z','2026-08-18T02:00:00Z')`,
		`INSERT INTO workspace_knowledge_revisions(knowledge_id,revision,supersedes_revision,title,content,created_by,created_at) VALUES('knowledge-1',1,0,'One','body','user-1','2026-08-18T01:00:00Z'),('knowledge-1',2,1,'Two','body','user-2','2026-08-18T02:00:00Z')`,
		`INSERT INTO workspace_knowledge_source_refs(knowledge_id,revision,ordinal,source_type,source_id,source_revision,citation) VALUES('knowledge-1',2,0,'acceptance_conclusion','issue-1','sha256:abc','Acceptance passed')`,
		`INSERT INTO workspace_knowledge_revisions(knowledge_id,revision,supersedes_revision,title,content,created_by,created_at) VALUES('foreign',1,0,'Foreign','body','user-9','2026-08-18T01:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := persistence.NewKnowledgeQueryRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	values, err := repository.ListGovernedKnowledge(context.Background(), "workspace-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || len(values[0].Revisions) != 2 || len(values[0].Revisions[1].SourceRefs) != 1 || values[0].Revisions[1].SourceRefs[0].Citation != "Acceptance passed" {
		t.Fatalf("values = %#v", values)
	}
}
