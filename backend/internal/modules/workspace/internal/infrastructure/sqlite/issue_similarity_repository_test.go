package sqlite_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

func TestIssueSimilarityRepositoryFindsNearNormalizedTitle(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "near-title", "workspace-1", 1, "WSP-1", "Alpha—Beta delivery", "", "todo", "2026-08-21T00:00:00Z")

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("Alpha beta"),
		Limit:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Issue.ID != "near-title" {
		t.Fatalf("results = %+v, want near-title", results)
	}
}

func TestIssueSimilarityRepositoryFindsDescriptionOverlap(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "description-overlap", "workspace-1", 1, "WSP-1", "Other work", "alpha beta delivery diagnosis", "todo", "2026-08-21T00:00:00Z")

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("Different request"),
		Description: application.NormalizeIssueSearchText("alpha beta delivery"),
		Limit:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Issue.ID != "description-overlap" {
		t.Fatalf("results = %+v, want description-overlap", results)
	}
}

func TestIssueSimilarityRepositoryFindsIdentifierCandidate(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "identifier", "workspace-1", 7, "WSP-7", "Other work", "", "todo", "2026-08-21T00:00:00Z")

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("wsp—7"),
		Limit:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Issue.ID != "identifier" {
		t.Fatalf("results = %+v, want identifier", results)
	}
}

func TestIssueSimilarityRepositoryBoostsSameProjectCandidate(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "same-project", "workspace-1", 1, "WSP-1", "Alpha beta delivery", "", "todo", "2026-08-20T00:00:00Z")
	seedSearchIssue(t, db, "other-project", "workspace-1", 2, "WSP-2", "Alpha beta delivery", "", "todo", "2026-08-21T00:00:00Z")
	if _, err := db.Exec(`UPDATE workspace_issues SET project_id='project-1' WHERE id='same-project'`); err != nil {
		t.Fatal(err)
	}

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("alpha beta"),
		ProjectID:   "project-1",
		Limit:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Issue.ID != "same-project" || !results[0].SameProject {
		t.Fatalf("results = %+v, want same-project first", results)
	}
}

func TestIssueSimilarityRepositoryIncludesClosedOnlyWhenRequested(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "closed", "workspace-1", 1, "WSP-1", "Alpha beta delivery", "", "done", "2026-08-21T00:00:00Z")

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	query := application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("alpha beta"),
		Limit:       50,
	}
	withoutClosed, _, err := repository.FindIssueSimilarityCandidates(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutClosed) != 0 {
		t.Fatalf("default results = %+v, want no closed candidates", withoutClosed)
	}
	query.IncludeClosed = true
	withClosed, _, err := repository.FindIssueSimilarityCandidates(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(withClosed) != 1 || withClosed[0].Issue.ID != "closed" || !withClosed[0].Closed {
		t.Fatalf("include closed results = %+v, want closed", withClosed)
	}
}

func TestIssueSimilarityRepositoryExplainsExactAndNearTitleScores(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "exact", "workspace-1", 1, "WSP-1", "Alpha beta", "", "todo", "2026-08-21T00:00:00Z")
	seedSearchIssue(t, db, "near", "workspace-1", 2, "WSP-2", "Alpha beta delivery", "", "todo", "2026-08-20T00:00:00Z")

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("alpha beta"),
		Limit:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Issue.ID != "exact" || results[0].ComponentScores["exact_normalized_title"] != 1 {
		t.Fatalf("exact result = %+v", results)
	}
	if results[1].Issue.ID != "near" || results[1].ComponentScores["title_terms"] != 1 || results[1].ComponentScores["exact_normalized_title"] != 0 || results[0].Score <= results[1].Score {
		t.Fatalf("near result = %+v", results)
	}
}

func TestIssueSimilarityRepositoryRanksHigherScoreBeforeNewerLowerScore(t *testing.T) {
	db := openIssueSearchDB(t)
	seedSearchIssue(t, db, "near-title", "workspace-1", 1, "WSP-1", "Alpha beta delivery", "", "todo", "2026-08-20T00:00:00Z")
	seedSearchIssue(t, db, "description-only", "workspace-1", 2, "WSP-2", "Unrelated title", "alpha beta delivery diagnosis", "todo", "2026-08-21T00:00:00Z")

	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1",
		Title:       application.NormalizeIssueSearchText("alpha beta"),
		Description: application.NormalizeIssueSearchText("alpha beta delivery"),
		Limit:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Issue.ID != "near-title" || results[0].Score <= results[1].Score {
		t.Fatalf("ranked results = %+v, want higher-scored near title first", results)
	}
}

func TestIssueSimilarityRepositoryNeverReadsMoreThanFiftyCandidates(t *testing.T) {
	db := openIssueSearchDB(t)
	for number := 1; number <= 51; number++ {
		seedSearchIssue(t, db,
			fmt.Sprintf("candidate-%d", number), "workspace-1", number, fmt.Sprintf("WSP-%d", number),
			"Alpha beta delivery", "", "todo", "2026-08-21T00:00:00Z",
		)
	}
	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), application.IssueSimilarityQuery{
		WorkspaceID: "workspace-1", Title: application.NormalizeIssueSearchText("alpha beta"), Limit: 51,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 50 {
		t.Fatalf("candidate count = %d, want maximum 50", len(results))
	}
}

func TestIssueSimilarityRepositoryTenThousandIssueLatencyBudget(t *testing.T) {
	db := openIssueSearchDB(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.Prepare(`INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,description,status,priority,
		creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at
	) VALUES(?,?,?,?,?,?,'todo','none','member','member-1',0,'{}','{}','[]','2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	for number := 1; number <= 10_000; number++ {
		title := fmt.Sprintf("Routine issue %d", number)
		if number%50 == 0 {
			title = fmt.Sprintf("Needle alpha %d", number)
		}
		if _, err := statement.Exec(fmt.Sprintf("issue-%05d", number), "workspace-1", number, fmt.Sprintf("WSP-%d", number), title, "bounded performance fixture"); err != nil {
			t.Fatal(err)
		}
	}
	if err := statement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewIssueSimilarityRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	query := application.IssueSimilarityQuery{WorkspaceID: "workspace-1", Title: application.NormalizeIssueSearchText("needle alpha"), Limit: 50}
	if results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), query); err != nil || len(results) != 50 {
		t.Fatalf("warm similarity results=%d error=%v", len(results), err)
	}
	if raceDetectorEnabled {
		t.Log("race instrumentation: correctness passed; latency budget is measured only by the non-instrumented gate")
		return
	}
	durations := make([]time.Duration, 20)
	for index := range durations {
		started := time.Now()
		if results, _, err := repository.FindIssueSimilarityCandidates(context.Background(), query); err != nil || len(results) != 50 {
			t.Fatalf("similarity %d results=%d error=%v", index, len(results), err)
		}
		durations[index] = time.Since(started)
	}
	sort.Slice(durations, func(left, right int) bool { return durations[left] < durations[right] })
	p50, p95 := durations[len(durations)/2], durations[18]
	t.Logf("10,000-Issue similarity latency p50=%s p95=%s", p50, p95)
	if p95 > 400*time.Millisecond {
		t.Fatalf("latency budget exceeded: p50=%s p95=%s", p50, p95)
	}
}
