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

func TestIssueSearchRepositoryRanksNormalizesPaginatesAndIsolates(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "identifier", "workspace-1", 1, "ALPHA-BETA", "Other", "", "todo", "2026-08-01T00:00:00Z")
	seedSearchIssue(t, db, "exact-title", "workspace-1", 2, "WSP-2", "Ａｌｐｈａ—ＢＥＴＡ", "", "todo", "2026-08-05T00:00:00Z")
	seedSearchIssue(t, db, "title-a", "workspace-1", 3, "WSP-3", "Beta delivery alpha", "", "todo", "2026-08-04T00:00:00Z")
	seedSearchIssue(t, db, "title-b", "workspace-1", 4, "WSP-4", "Alpha then beta", "", "todo", "2026-08-04T00:00:00Z")
	seedSearchIssue(t, db, "description", "workspace-1", 5, "WSP-5", "Other", "Alpha plus beta lives here", "todo", "2026-08-06T00:00:00Z")
	seedSearchIssue(t, db, "closed", "workspace-1", 6, "WSP-6", "Alpha beta closed", "", "done", "2026-08-07T00:00:00Z")
	seedSearchIssue(t, db, "foreign", "workspace-2", 1, "TWO-1", "Alpha beta foreign", "", "todo", "2026-08-08T00:00:00Z")

	repository, err := persistence.NewIssueSearchRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	query := application.IssueSearchQuery{WorkspaceID: "workspace-1", Phrase: "alpha beta", Terms: []string{"alpha", "beta"}, Limit: 2}
	first, total, err := repository.SearchIssues(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	assertSearchIDs(t, first, "identifier", "exact-title")
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	query.Offset = 2
	query.Limit = 10
	second, total, err := repository.SearchIssues(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	assertSearchIDs(t, second, "title-a", "title-b", "description")
	if total != 5 || second[2].MatchSource != "description" || second[2].DescriptionSnippet == "" {
		t.Fatalf("second page = %+v total=%d", second, total)
	}
	query.IncludeClosed = true
	all, total, err := repository.SearchIssues(context.Background(), query)
	if err != nil || total != 6 {
		t.Fatalf("include closed total=%d error=%v", total, err)
	}
	assertSearchIDs(t, all, "closed", "title-a", "title-b", "description")
}

func TestIssueSearchRepositorySupportsChineseIdentifierNumberSyncRestartAndCancellation(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-search.db")
	db := openIssueSearchDBAt(t, databasePath)
	seedSearchIssue(t, db, "zh", "workspace-1", 41, "WSP-41", "修复咖啡机搜索", "检查中文索引", "todo", "2026-08-01T00:00:00Z")
	repository, err := persistence.NewIssueSearchRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	assertSingleSearch(t, repository, "workspace-1", "咖啡机", "zh")
	assertSingleSearch(t, repository, "workspace-1", "ｗｓｐ－４１", "zh")
	assertSingleSearch(t, repository, "workspace-1", "41", "zh")

	if _, err := db.Exec(`UPDATE workspace_issues SET title='新的中文标题',updated_at='2026-08-02T00:00:00Z' WHERE id='zh'`); err != nil {
		t.Fatal(err)
	}
	assertSingleSearch(t, repository, "workspace-1", "新的 中文", "zh")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := sql.Open("sqlite", "file:"+filepath.ToSlash(databasePath))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	repository, err = persistence.NewIssueSearchRepository(persistence.Config{DB: reopened})
	if err != nil {
		t.Fatal(err)
	}
	assertSingleSearch(t, repository, "workspace-1", "新的中文", "zh")

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = repository.SearchIssues(cancelled, application.IssueSearchQuery{WorkspaceID: "workspace-1", Phrase: "中文", Terms: []string{"中文"}, Limit: 20})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled error = %v", err)
	}
	if _, err := reopened.Exec(`DELETE FROM workspace_issues WHERE id='zh'`); err != nil {
		t.Fatal(err)
	}
	results, total, err := repository.SearchIssues(context.Background(), application.IssueSearchQuery{WorkspaceID: "workspace-1", Phrase: "中文", Terms: []string{"中文"}, Limit: 20})
	if err != nil || total != 0 || len(results) != 0 {
		t.Fatalf("deleted search = %+v total=%d error=%v", results, total, err)
	}
}

func TestIssueSearchRepositoryTenThousandIssueLatencyBudget(t *testing.T) {
	db := openIssueSearchDB(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.Prepare(`INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,description,status,priority,
		creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at
	) VALUES(?,?,?,?,?,?,'todo','none','member','member-1',0,'{}','{}','[]','2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 10_000; index++ {
		title := fmt.Sprintf("Routine Issue %d", index)
		if index%50 == 0 {
			title = fmt.Sprintf("Needle Alpha %d", index)
		}
		if _, err := statement.Exec(fmt.Sprintf("issue-%05d", index), "workspace-1", index, fmt.Sprintf("WSP-%d", index), title, "bounded performance fixture"); err != nil {
			t.Fatal(err)
		}
	}
	if err := statement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewIssueSearchRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	query := application.IssueSearchQuery{WorkspaceID: "workspace-1", Phrase: "needle alpha", Terms: []string{"needle", "alpha"}, Limit: 50}
	if _, total, err := repository.SearchIssues(context.Background(), query); err != nil || total != 200 {
		t.Fatalf("warm search total=%d error=%v", total, err)
	}
	if raceDetectorEnabled {
		t.Log("race instrumentation: correctness passed; latency budget is measured only by the non-instrumented gate")
		return
	}
	durations := make([]time.Duration, 20)
	for index := range durations {
		started := time.Now()
		if _, total, err := repository.SearchIssues(context.Background(), query); err != nil || total != 200 {
			t.Fatalf("search %d total=%d error=%v", index, total, err)
		}
		durations[index] = time.Since(started)
	}
	sort.Slice(durations, func(left, right int) bool { return durations[left] < durations[right] })
	p50, p95 := durations[len(durations)/2], durations[18]
	t.Logf("10,000-Issue search latency p50=%s p95=%s", p50, p95)
	if p50 > 100*time.Millisecond || p95 > 250*time.Millisecond {
		t.Fatalf("latency budget exceeded: p50=%s p95=%s", p50, p95)
	}
}

func openIssueSearchDB(t *testing.T) *sql.DB {
	t.Helper()
	return openIssueSearchDBAt(t, filepath.Join(t.TempDir(), "issue-search.db"))
}

func openIssueSearchDBAt(t *testing.T, path string) *sql.DB {
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

func seedSearchIssue(t *testing.T, db *sql.DB, id, workspaceID string, number int, identifier, title, description, status, updated string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,description,status,priority,
		creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,'none','member','member-1',0,'{}','{}','[]',?,?)`,
		id, workspaceID, number, identifier, title, description, status, updated, updated)
	if err != nil {
		t.Fatal(err)
	}
}

func assertSingleSearch(t *testing.T, repository application.IssueSearchRepository, workspaceID, raw, id string) {
	t.Helper()
	phrase := application.NormalizeIssueSearchText(raw)
	results, total, err := repository.SearchIssues(context.Background(), application.IssueSearchQuery{WorkspaceID: workspaceID, Phrase: phrase, Terms: splitSearchTerms(phrase), Limit: 20})
	if err != nil || total != 1 || len(results) != 1 || results[0].Issue.ID != id {
		t.Fatalf("search %q = %+v total=%d error=%v", raw, results, total, err)
	}
}

func splitSearchTerms(value string) []string {
	return strings.Fields(value)
}

func assertSearchIDs(t *testing.T, results []application.IssueSearchResult, ids ...string) {
	t.Helper()
	if len(results) != len(ids) {
		t.Fatalf("result count = %d, want %d: %+v", len(results), len(ids), results)
	}
	for index, id := range ids {
		if results[index].Issue.ID != id {
			t.Fatalf("result %d = %s, want %s", index, results[index].Issue.ID, id)
		}
	}
}

func ExampleNormalizeIssueSearchText() {
	fmt.Println(application.NormalizeIssueSearchText(" Ｃａｆé—API  搜索 "))
	// Output: café api 搜索
}
