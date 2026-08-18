package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestProjectSearchRepositoryRanksNormalizesPaginatesAndIsolates(t *testing.T) {
	db := openProjectSearchDB(t)
	seedSearchProject(t, db, "exact", "workspace-1", "Ａｌｐｈａ—ＢＥＴＡ", "", "planned", "2026-08-01T00:00:00Z")
	seedSearchProject(t, db, "title-a", "workspace-1", "Beta delivery alpha", "", "planned", "2026-08-04T00:00:00Z")
	seedSearchProject(t, db, "title-b", "workspace-1", "Alpha then beta", "", "planned", "2026-08-04T00:00:00Z")
	seedSearchProject(t, db, "description", "workspace-1", "Other", "Alpha plus beta lives here", "planned", "2026-08-06T00:00:00Z")
	seedSearchProject(t, db, "closed", "workspace-1", "Alpha beta closed", "", "completed", "2026-08-07T00:00:00Z")
	seedSearchProject(t, db, "foreign", "workspace-2", "Alpha beta foreign", "", "planned", "2026-08-08T00:00:00Z")

	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	query := application.ProjectSurfaceSearchQuery{WorkspaceID: "workspace-1", Phrase: "alpha beta", Terms: []string{"alpha", "beta"}, Limit: 2}
	first, total, err := repository.SearchProjects(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectSearchIDs(t, first, "exact", "title-a")
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	query.Offset, query.Limit = 2, 10
	second, total, err := repository.SearchProjects(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectSearchIDs(t, second, "title-b", "description")
	if total != 4 || second[1].MatchSource != "description" || second[1].MatchedSnippet == "" {
		t.Fatalf("second page = %+v total=%d", second, total)
	}
	query.IncludeClosed = true
	query.Offset = 0
	all, total, err := repository.SearchProjects(context.Background(), query)
	if err != nil || total != 5 {
		t.Fatalf("include closed total=%d error=%v", total, err)
	}
	assertProjectSearchIDs(t, all, "exact", "closed", "title-a", "title-b", "description")
}

func TestProjectSearchRepositorySupportsChineseSyncRestartAndCancellation(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "project-search.db")
	db := openProjectSearchDBAt(t, databasePath)
	seedSearchProject(t, db, "zh", "workspace-1", "修复咖啡机搜索", "检查中文索引", "planned", "2026-08-01T00:00:00Z")
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	assertSingleProjectSearch(t, repository, "workspace-1", "咖啡机", "zh")
	if _, err := db.Exec(`UPDATE workspace_projects SET name='新的中文标题',updated_at='2026-08-02T00:00:00Z' WHERE id='zh'`); err != nil {
		t.Fatal(err)
	}
	assertSingleProjectSearch(t, repository, "workspace-1", "新的 中文", "zh")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databasePath))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	repository, err = persistence.NewProjectSurfaceRepository(persistence.Config{DB: reopened})
	if err != nil {
		t.Fatal(err)
	}
	assertSingleProjectSearch(t, repository, "workspace-1", "新的中文", "zh")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = repository.SearchProjects(cancelled, application.ProjectSurfaceSearchQuery{WorkspaceID: "workspace-1", Phrase: "中文", Terms: []string{"中文"}, Limit: 20})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled error = %v", err)
	}
	if _, err := reopened.Exec(`DELETE FROM workspace_projects WHERE id='zh'`); err != nil {
		t.Fatal(err)
	}
	results, total, err := repository.SearchProjects(context.Background(), application.ProjectSurfaceSearchQuery{WorkspaceID: "workspace-1", Phrase: "中文", Terms: []string{"中文"}, Limit: 20})
	if err != nil || total != 0 || len(results) != 0 {
		t.Fatalf("deleted search = %+v total=%d error=%v", results, total, err)
	}
}

func TestProjectSearchRepositoryTwoThousandProjectLatencyBudget(t *testing.T) {
	db := openProjectSearchDB(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.Prepare(`INSERT INTO workspace_projects(id,workspace_id,name,description,status,asset_ids,created_at,updated_at) VALUES(?,?,?,?,'planned','[]','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 2_000; index++ {
		title := fmt.Sprintf("Routine Project %d", index)
		if index%20 == 0 {
			title = fmt.Sprintf("Needle Alpha %d", index)
		}
		if _, err := statement.Exec(fmt.Sprintf("project-%05d", index), "workspace-1", title, "bounded performance fixture"); err != nil {
			t.Fatal(err)
		}
	}
	_ = statement.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewProjectSurfaceRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	query := application.ProjectSurfaceSearchQuery{WorkspaceID: "workspace-1", Phrase: "needle alpha", Terms: []string{"needle", "alpha"}, Limit: 50}
	if _, total, err := repository.SearchProjects(context.Background(), query); err != nil || total != 100 {
		t.Fatalf("warm search total=%d error=%v", total, err)
	}
	if raceDetectorEnabled {
		t.Log("race instrumentation: correctness passed; latency budget is measured only by the non-instrumented gate")
		return
	}
	durations := make([]time.Duration, 20)
	for index := range durations {
		started := time.Now()
		if _, total, err := repository.SearchProjects(context.Background(), query); err != nil || total != 100 {
			t.Fatalf("search %d total=%d error=%v", index, total, err)
		}
		durations[index] = time.Since(started)
	}
	sort.Slice(durations, func(left, right int) bool { return durations[left] < durations[right] })
	p50, p95 := durations[len(durations)/2], durations[18]
	t.Logf("2,000-Project search latency p50=%s p95=%s", p50, p95)
	if p50 > 100*time.Millisecond || p95 > 250*time.Millisecond {
		t.Fatalf("latency budget exceeded: p50=%s p95=%s", p50, p95)
	}
}

func openProjectSearchDB(t *testing.T) *sql.DB {
	t.Helper()
	return openProjectSearchDBAt(t, filepath.Join(t.TempDir(), "project-search.db"))
}

func openProjectSearchDBAt(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedSearchProject(t *testing.T, db *sql.DB, id, workspaceID, title, description, status, updated string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,description,status,asset_ids,created_at,updated_at) VALUES(?,?,?,?,?,'[]',?,?)`, id, workspaceID, title, description, status, updated, updated); err != nil {
		t.Fatal(err)
	}
}

func assertSingleProjectSearch(t *testing.T, repository *persistence.ProjectSurfaceRepository, workspaceID, raw, id string) {
	t.Helper()
	phrase := application.NormalizeIssueSearchText(raw)
	results, total, err := repository.SearchProjects(context.Background(), application.ProjectSurfaceSearchQuery{WorkspaceID: workspaceID, Phrase: phrase, Terms: strings.Fields(phrase), Limit: 20})
	if err != nil || total != 1 || len(results) != 1 || results[0].Project.ID != id {
		t.Fatalf("search %q = %+v total=%d error=%v", raw, results, total, err)
	}
}

func assertProjectSearchIDs(t *testing.T, results []application.ProjectSurfaceSearchResult, ids ...string) {
	t.Helper()
	if len(results) != len(ids) {
		t.Fatalf("result count = %d, want %d: %+v", len(results), len(ids), results)
	}
	for index, id := range ids {
		if results[index].Project.ID != id {
			t.Fatalf("result %d = %s, want %s", index, results[index].Project.ID, id)
		}
	}
}
