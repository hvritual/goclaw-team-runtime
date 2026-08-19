package sqlite_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	moderncSQLite "modernc.org/sqlite"
)

func TestProjectRequirementRepositoryDerivesCurrentAndEffectiveCoverageFailClosed(t *testing.T) {
	db := openProjectRequirementDB(t)
	content := `{"problem_statement":"Ship safely","goals":[{"key":"goal-unlinked","text":"No work"},{"key":"goal-open","text":"Open work"},{"key":"goal-removed","text":"Removed work"}],"in_scope":[{"key":"scope-done","text":"Implemented work"}],"out_of_scope":[],"constraints":[{"key":"constraint-accepted","text":"Accepted work"}],"acceptance_criteria":[{"key":"acceptance-mixed","text":"All work must finish"}],"dependencies":[]}`
	for _, statement := range []string{
		`INSERT INTO workspace_requirement_baselines(
			id,workspace_id,project_id,status,current_revision,approved_revision,effective_revision,
			review_origin,latest_content_author,submitted_by,submitted_at,approved_by,approved_at,
			frozen_by,frozen_at,retired_by,retired_at,created_at,updated_at
		) VALUES('baseline-coverage','workspace-1','project-1','retired',4,2,2,NULL,'lead-1',
			NULL,NULL,'owner-1','2026-08-19T00:02:00Z','owner-1','2026-08-19T00:03:00Z',
			'owner-1','2026-08-19T00:04:00Z','2026-08-19T00:00:00Z','2026-08-19T00:04:00Z')`,
		`INSERT INTO workspace_requirement_revisions(
			baseline_id,revision,content_json,status,action,change_summary,actor_id,
			submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
		) VALUES
			('baseline-coverage',1,'` + content + `','draft','create','Create','lead-1',NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-19T00:00:00Z'),
			('baseline-coverage',2,'` + content + `','frozen','freeze','Freeze','owner-1',NULL,NULL,'owner-1','2026-08-19T00:02:00Z','owner-1','2026-08-19T00:02:00Z','2026-08-19T00:02:00Z'),
			('baseline-coverage',3,'` + content + `','frozen','unlink_issue','Unlink Issue','lead-1',NULL,NULL,'owner-1','2026-08-19T00:02:00Z','owner-1','2026-08-19T00:02:00Z','2026-08-19T00:03:00Z'),
			('baseline-coverage',4,'` + content + `','retired','retire','Retire','owner-1',NULL,NULL,'owner-1','2026-08-19T00:02:00Z','owner-1','2026-08-19T00:02:00Z','2026-08-19T00:04:00Z')`,
		`INSERT INTO workspace_issues(
			id,workspace_id,number,identifier,title,description,status,priority,creator_type,creator_id,
			project_id,metadata,properties,asset_ids,created_at,updated_at
		) VALUES
			('issue-open','workspace-1',2,'ONE-2','Open','','in_progress','none','member','lead-1','project-1','{}','{}','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z'),
			('issue-done','workspace-1',3,'ONE-3','Done conditional','','done','none','member','lead-1','project-1','{}','{}','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z'),
			('issue-accepted','workspace-1',4,'ONE-4','Accepted','','done','none','member','lead-1','project-1','{}','{}','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z'),
			('issue-removed','workspace-1',5,'ONE-5','Removed accepted','','done','none','member','lead-1','project-1','{}','{}','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_requirement_issue_links(
			workspace_id,project_id,baseline_id,requirement_key,issue_id,linked_revision,
			unlinked_revision,linked_by,linked_at,unlinked_by,unlinked_at
		) VALUES
			('workspace-1','project-1','baseline-coverage','goal-unlinked','issue-deleted',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL),
			('workspace-1','project-1','baseline-coverage','goal-open','issue-open',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL),
			('workspace-1','project-1','baseline-coverage','goal-removed','issue-removed',1,3,'lead-1','2026-08-19T00:00:00Z','lead-1','2026-08-19T00:03:00Z'),
			('workspace-1','project-1','baseline-coverage','scope-done','issue-done',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL),
			('workspace-1','project-1','baseline-coverage','constraint-accepted','issue-accepted',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL),
			('workspace-1','project-1','baseline-coverage','acceptance-mixed','issue-open',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL),
			('workspace-1','project-1','baseline-coverage','acceptance-mixed','issue-accepted',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL)`,
		`INSERT INTO workspace_issue_acceptance_conclusions(
			id,workspace_id,issue_id,result,rationale,evidence_refs,actor_id,created_at,updated_at
		) VALUES
			('conclusion-done-old','workspace-1','issue-done','accepted','Old acceptance','[]','owner-1','2026-08-19T00:01:00Z','2026-08-19T00:01:00Z'),
			('conclusion-done-new','workspace-1','issue-done','conditional','Latest condition','[]','owner-1','2026-08-19T00:02:00Z','2026-08-19T00:02:00Z'),
			('conclusion-accepted','workspace-1','issue-accepted','accepted','Accepted','[]','owner-1','2026-08-19T00:01:00Z','2026-08-19T00:01:00Z'),
			('conclusion-removed','workspace-1','issue-removed','accepted','Accepted','[]','owner-1','2026-08-19T00:01:00Z','2026-08-19T00:01:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := repository.ReadProjectRequirementCoverage(
		context.Background(), "workspace-1", "project-1",
		contract.WorkspaceActor{Type: "member", ID: "ordinary-1"},
	)
	if err != nil {
		t.Fatalf("ReadProjectRequirementCoverage() error = %v", err)
	}
	if coverage.BaselineStatus == nil || *coverage.BaselineStatus != "retired" {
		t.Fatalf("baseline status = %#v", coverage.BaselineStatus)
	}
	if coverage.Current == nil || coverage.Current.Revision != 4 || coverage.Current.State != "retired" {
		t.Fatalf("current snapshot = %#v", coverage.Current)
	}
	if got := []int{coverage.Current.Total, coverage.Current.Linked, coverage.Current.Implemented, coverage.Current.Accepted, coverage.Current.Unlinked}; !equalCoverageCounts(got, []int{6, 4, 2, 1, 2}) {
		t.Fatalf("current counts = %v", got)
	}
	currentStages := []string{"unlinked", "linked", "unlinked", "implemented", "accepted", "linked"}
	for index, want := range currentStages {
		if coverage.Current.Items[index].Stage != want {
			t.Fatalf("current item %d = %#v, want stage %q", index, coverage.Current.Items[index], want)
		}
	}
	if coverage.Current.Items[3].Issues[0].AcceptanceResult == nil || *coverage.Current.Items[3].Issues[0].AcceptanceResult != "conditional" {
		t.Fatalf("latest conditional issue = %#v", coverage.Current.Items[3].Issues)
	}
	if issues := coverage.Current.Items[5].Issues; len(issues) != 2 || issues[0].Identifier != "ONE-2" || issues[1].Identifier != "ONE-4" {
		t.Fatalf("deterministic mixed issues = %#v", issues)
	}
	if coverage.Effective == nil || coverage.Effective.Revision != 2 || coverage.Effective.State != "frozen" {
		t.Fatalf("effective snapshot = %#v", coverage.Effective)
	}
	if got := []int{coverage.Effective.Total, coverage.Effective.Linked, coverage.Effective.Implemented, coverage.Effective.Accepted, coverage.Effective.Unlinked}; !equalCoverageCounts(got, []int{6, 5, 3, 2, 1}) {
		t.Fatalf("effective counts = %v", got)
	}
	if coverage.Effective.Items[2].Stage != "accepted" || len(coverage.Effective.Items[2].Issues) != 1 {
		t.Fatalf("effective removed-link item = %#v", coverage.Effective.Items[2])
	}

	if _, err = db.Exec(`UPDATE workspace_projects SET status='completed' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ReadProjectRequirementCoverage(context.Background(), "workspace-1", "project-1", contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}); err != nil {
		t.Fatalf("completed-project coverage read error = %v", err)
	}
	if _, err = repository.ReadProjectRequirementCoverage(context.Background(), "workspace-1", "project-1", contract.WorkspaceActor{Type: "member", ID: "removed-1"}); !errors.Is(err, contract.ErrActorOutsideWorkspace) {
		t.Fatalf("removed member error = %v, want ErrActorOutsideWorkspace", err)
	}
}

func TestProjectRequirementRepositoryReturnsExplicitEmptyCoverageWithoutBaseline(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := repository.ReadProjectRequirementCoverage(
		context.Background(), "workspace-1", "project-1",
		contract.WorkspaceActor{Type: "member", ID: "ordinary-1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.BaselineStatus != nil || coverage.Current != nil || coverage.Effective != nil {
		t.Fatalf("empty coverage = %#v", coverage)
	}
}

func TestProjectRequirementRepositoryRejectsPersistedContentOutsideDomainInvariant(t *testing.T) {
	validContent := marshalProjectRequirementTestContent(t, projectRequirementTestContent("Valid content"))
	oversizedContent := marshalProjectRequirementTestContent(t, requirementDomain.Content{
		ProblemStatement: "Oversized",
		Goals:            []requirementDomain.Item{{Key: "goal-1", Text: strings.Repeat("x", 2001)}},
	})
	tests := []struct {
		name             string
		currentContent   string
		effectiveContent string
	}{
		{name: "empty current content", currentContent: `{}`, effectiveContent: validContent},
		{name: "empty effective content", currentContent: validContent, effectiveContent: `{}`},
		{name: "invalid traceability key", currentContent: `{"problem_statement":"Invalid key","goals":[{"key":"bad key","text":"Goal"}]}`, effectiveContent: validContent},
		{name: "duplicate traceability key", currentContent: `{"problem_statement":"Duplicate","goals":[{"key":"same","text":"Goal"}],"in_scope":[{"key":"same","text":"Scope"}]}`, effectiveContent: validContent},
		{name: "oversized item", currentContent: oversizedContent, effectiveContent: validContent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			if _, err := db.Exec(`INSERT INTO workspace_requirement_baselines(
				id,workspace_id,project_id,status,current_revision,approved_revision,effective_revision,
				review_origin,latest_content_author,submitted_by,submitted_at,approved_by,approved_at,
				frozen_by,frozen_at,retired_by,retired_at,created_at,updated_at
			) VALUES('baseline-invalid-content','workspace-1','project-1','changed',2,1,1,'changed','lead-1',
				NULL,NULL,'owner-1','2026-08-19T00:01:00Z','owner-1','2026-08-19T00:01:00Z',
				NULL,NULL,'2026-08-19T00:00:00Z','2026-08-19T00:02:00Z')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO workspace_requirement_revisions(
				baseline_id,revision,content_json,status,action,change_summary,actor_id,
				submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
			) VALUES
				('baseline-invalid-content',1,?,'frozen','freeze','Frozen','owner-1',NULL,NULL,'owner-1','2026-08-19T00:01:00Z','owner-1','2026-08-19T00:01:00Z','2026-08-19T00:01:00Z'),
				('baseline-invalid-content',2,?,'changed','material_change','Changed','lead-1',NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-19T00:02:00Z')`,
				test.effectiveContent, test.currentContent); err != nil {
				t.Fatal(err)
			}

			repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			coverage, err := repository.ReadProjectRequirementCoverage(
				context.Background(), "workspace-1", "project-1",
				contract.WorkspaceActor{Type: "member", ID: "ordinary-1"},
			)
			if err == nil {
				t.Fatalf("ReadProjectRequirementCoverage() error = nil, coverage = %#v", coverage)
			}
			if coverage.BaselineStatus != nil || coverage.Current != nil || coverage.Effective != nil {
				t.Fatalf("failed coverage returned partial projection = %#v", coverage)
			}
		})
	}
}

func TestProjectRequirementRepositoryRejectsActiveIssueLinkOwnershipDrift(t *testing.T) {
	db := openProjectRequirementDB(t)
	content := marshalProjectRequirementTestContent(t, requirementDomain.Content{
		ProblemStatement: "Keep ownership local",
		Goals:            []requirementDomain.Item{{Key: "goal-1", Text: "Local goal"}},
	})
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES
			('workspace-2','Workspace Two','workspace-two','TWO','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,asset_ids,created_at,updated_at) VALUES
			('project-2','workspace-2','Foreign Project','in_progress','none','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_issues(
			id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at
		) VALUES('foreign-issue','workspace-2',1,'TWO-1','Foreign secret title','done','none','member','foreign-owner','project-2','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_requirement_baselines(
			id,workspace_id,project_id,status,current_revision,approved_revision,effective_revision,
			review_origin,latest_content_author,submitted_by,submitted_at,approved_by,approved_at,
			frozen_by,frozen_at,retired_by,retired_at,created_at,updated_at
		) VALUES('baseline-link-drift','workspace-1','project-1','draft',1,NULL,NULL,NULL,'lead-1',
			NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspace_requirement_revisions(
		baseline_id,revision,content_json,status,action,change_summary,actor_id,
		submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
	) VALUES('baseline-link-drift',1,?,'draft','create','Create','lead-1',NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-19T00:00:00Z')`, content); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace_requirement_issue_links(
		workspace_id,project_id,baseline_id,requirement_key,issue_id,linked_revision,
		unlinked_revision,linked_by,linked_at,unlinked_by,unlinked_at
	) VALUES('workspace-2','project-2','baseline-link-drift','goal-1','foreign-issue',1,NULL,'lead-1','2026-08-19T00:00:00Z',NULL,NULL)`); err != nil {
		t.Fatal(err)
	}

	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := repository.ReadProjectRequirementCoverage(
		context.Background(), "workspace-1", "project-1",
		contract.WorkspaceActor{Type: "member", ID: "ordinary-1"},
	)
	if err == nil {
		t.Fatalf("ReadProjectRequirementCoverage() error = nil, leaked coverage = %#v", coverage)
	}
	if strings.Contains(err.Error(), "TWO-1") || strings.Contains(err.Error(), "Foreign secret title") {
		t.Fatalf("ownership error leaked foreign Issue detail: %v", err)
	}
	if coverage.BaselineStatus != nil || coverage.Current != nil || coverage.Effective != nil {
		t.Fatalf("failed coverage returned partial projection = %#v", coverage)
	}
}

func TestProjectRequirementCoverageQueryCountIsBoundedBySnapshotCount(t *testing.T) {
	oneItem := requirementDomain.Content{
		ProblemStatement: "One item",
		Goals:            []requirementDomain.Item{{Key: "goal-001", Text: "Goal 1"}},
	}
	oneHundredItems := requirementDomain.Content{ProblemStatement: "One hundred items"}
	for index := 1; index <= 100; index++ {
		oneHundredItems.Goals = append(oneHundredItems.Goals, requirementDomain.Item{
			Key: fmt.Sprintf("goal-%03d", index), Text: fmt.Sprintf("Goal %d", index),
		})
	}
	oneItemJSON := marshalProjectRequirementTestContent(t, oneItem)
	oneHundredItemsJSON := marshalProjectRequirementTestContent(t, oneHundredItems)
	databasePath := filepath.Join(t.TempDir(), "project-requirements-counted.db")
	seedDB := openProjectRequirementDBAtPath(t, "sqlite", databasePath)
	if _, err := seedDB.Exec(`INSERT INTO workspace_requirement_baselines(
		id,workspace_id,project_id,status,current_revision,approved_revision,effective_revision,
		review_origin,latest_content_author,submitted_by,submitted_at,approved_by,approved_at,
		frozen_by,frozen_at,retired_by,retired_at,created_at,updated_at
	) VALUES('baseline-query-bound','workspace-1','project-1','changed',2,1,1,'changed','lead-1',
		NULL,NULL,'owner-1','2026-08-19T00:01:00Z','owner-1','2026-08-19T00:01:00Z',
		NULL,NULL,'2026-08-19T00:00:00Z','2026-08-19T00:02:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := seedDB.Exec(`INSERT INTO workspace_requirement_revisions(
		baseline_id,revision,content_json,status,action,change_summary,actor_id,
		submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
	) VALUES
		('baseline-query-bound',1,?,'frozen','freeze','Frozen','owner-1',NULL,NULL,'owner-1','2026-08-19T00:01:00Z','owner-1','2026-08-19T00:01:00Z','2026-08-19T00:01:00Z'),
		('baseline-query-bound',2,?,'changed','material_change','Changed','lead-1',NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-19T00:02:00Z')`,
		oneItemJSON, oneItemJSON); err != nil {
		t.Fatal(err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatal(err)
	}

	counter := &projectRequirementSQLQueryCounter{}
	driverName := fmt.Sprintf("project-requirement-counting-sqlite-%d", projectRequirementCountingDriverSequence.Add(1))
	sql.Register(driverName, &projectRequirementCountingDriver{delegate: &moderncSQLite.Driver{}, counter: counter})
	db, err := sql.Open(driverName, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	read := func() int64 {
		t.Helper()
		counter.Reset()
		coverage, readErr := repository.ReadProjectRequirementCoverage(
			context.Background(), "workspace-1", "project-1",
			contract.WorkspaceActor{Type: "member", ID: "ordinary-1"},
		)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if coverage.Current == nil || coverage.Effective == nil {
			t.Fatalf("coverage snapshots = %#v", coverage)
		}
		return counter.Load()
	}

	oneItemQueries := read()
	if _, err = db.Exec(`UPDATE workspace_requirement_revisions SET content_json=? WHERE baseline_id='baseline-query-bound'`, oneHundredItemsJSON); err != nil {
		t.Fatal(err)
	}
	oneHundredItemQueries := read()
	if oneItemQueries != 8 || oneHundredItemQueries != 8 {
		t.Fatalf("coverage query count = one item %d, one hundred items %d, want constant bound 8", oneItemQueries, oneHundredItemQueries)
	}
}

func equalCoverageCounts(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func TestProjectRequirementRepositoryCreatesAndReplaysOneAuthorizedBaseline(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	command := application.ProjectRequirementSave{
		BaselineID:       "baseline-1",
		WorkspaceID:      "workspace-1",
		ProjectID:        "project-1",
		ExpectedRevision: 0,
		Content:          projectRequirementTestContent("Initial baseline"),
		ChangeSummary:    "Initial baseline",
		IdempotencyKey:   "create-key",
		RequestHash:      strings.Repeat("a", 64),
		Actor:            contract.WorkspaceActor{Type: "member", ID: "owner-1"},
		OccurredAt:       time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC),
	}
	created, err := repository.SaveProjectRequirement(context.Background(), command)
	if err != nil {
		t.Fatalf("SaveProjectRequirement(create) error = %v", err)
	}
	if created.Baseline == nil || created.Baseline.ID != "baseline-1" || created.Baseline.Status != "draft" || created.Baseline.CurrentRevision != 1 {
		t.Fatalf("created baseline = %#v", created.Baseline)
	}
	if created.CurrentContent == nil || created.CurrentContent.ProblemStatement != "Initial baseline" {
		t.Fatalf("created current content = %#v", created.CurrentContent)
	}

	replayedCommand := command
	replayedCommand.BaselineID = "ignored-retry-id"
	replayed, err := repository.SaveProjectRequirement(context.Background(), replayedCommand)
	if err != nil {
		t.Fatalf("SaveProjectRequirement(replay) error = %v", err)
	}
	if replayed.Baseline == nil || replayed.Baseline.ID != "baseline-1" || replayed.Baseline.CurrentRevision != 1 {
		t.Fatalf("replayed baseline = %#v", replayed.Baseline)
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_baselines", 1)
	assertProjectRequirementRowCount(t, db, "workspace_requirement_revisions", 1)
	assertProjectRequirementRowCount(t, db, "workspace_mutation_idempotency", 1)
	assertProjectRequirementRowCount(t, db, "workspace_audit_entries", 1)
	assertProjectRequirementRowCount(t, db, "workspace_outbox_events", 1)

	conflictCommand := command
	conflictCommand.RequestHash = strings.Repeat("b", 64)
	if _, err := repository.SaveProjectRequirement(context.Background(), conflictCommand); !errors.Is(err, contract.ErrIdempotencyConflict) {
		t.Fatalf("SaveProjectRequirement(conflict) error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestProjectRequirementRepositoryDeniesUnprivilegedCreateBeforeWriting(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID:       "baseline-1",
		WorkspaceID:      "workspace-1",
		ProjectID:        "project-1",
		ExpectedRevision: 0,
		Content:          projectRequirementTestContent("Denied baseline"),
		ChangeSummary:    "Denied baseline",
		IdempotencyKey:   "denied-key",
		RequestHash:      strings.Repeat("c", 64),
		Actor:            contract.WorkspaceActor{Type: "member", ID: "ordinary-1"},
		OccurredAt:       time.Date(2026, 8, 19, 10, 1, 0, 0, time.UTC),
	})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("SaveProjectRequirement(denied) error = %v", err)
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_baselines", 0)
	assertProjectRequirementRowCount(t, db, "workspace_requirement_revisions", 0)
	assertProjectRequirementRowCount(t, db, "workspace_mutation_idempotency", 0)
	assertProjectRequirementRowCount(t, db, "workspace_audit_entries", 0)
	assertProjectRequirementRowCount(t, db, "workspace_outbox_events", 0)
}

func TestProjectRequirementRepositoryRereadsLiveAuthorityBeforeMutation(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		change  string
		wantErr error
	}{
		{name: "removed membership", change: "membership", wantErr: contract.ErrActorOutsideWorkspace},
		{name: "reassigned project lead", change: "lead", wantErr: contract.ErrWorkspacePermissionDenied},
		{name: "removed project editor grant", change: "grant", wantErr: contract.ErrWorkspacePermissionDenied},
		{name: "completed project", change: "completed", wantErr: contract.ErrWorkspacePermissionDenied},
		{name: "cancelled project", change: "cancelled", wantErr: contract.ErrWorkspacePermissionDenied},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 8, 19, 10, 5, 0, 0, time.UTC)
			owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
			actor := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
			if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
				BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
				Content: projectRequirementTestContent("Initial authority proof"), ChangeSummary: "Initial authority proof",
				IdempotencyKey: "live-authority-key", RequestHash: strings.Repeat("9", 64), Actor: owner, OccurredAt: now,
			}); err != nil {
				t.Fatal(err)
			}

			if testCase.change == "grant" {
				actor = contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}
				if _, err = repository.ReplaceProjectRequirementAccess(context.Background(), application.ProjectRequirementAccessReplace{
					WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
					Grants: []application.ProjectRequirementGrantChange{{MemberID: "ordinary-member", GrantKind: "project_editor"}},
					Actor:  owner, OccurredAt: now.Add(time.Minute),
				}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
				WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 1,
				Content: projectRequirementTestContent("Authorized before live change"), ChangeSummary: "Authorized before live change",
				Actor: actor, OccurredAt: now.Add(2 * time.Minute),
			}); err != nil {
				t.Fatalf("SaveProjectRequirement(before %s) error = %v", testCase.change, err)
			}

			switch testCase.change {
			case "membership":
				_, err = db.Exec(`DELETE FROM auth_members WHERE workspace_id='workspace-1' AND user_id='lead-1'`)
			case "lead":
				_, err = db.Exec(`UPDATE workspace_projects SET lead_id='ordinary-member' WHERE workspace_id='workspace-1' AND id='project-1'`)
			case "grant":
				_, err = repository.ReplaceProjectRequirementAccess(context.Background(), application.ProjectRequirementAccessReplace{
					WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 1,
					Grants: nil, Actor: owner, OccurredAt: now.Add(3 * time.Minute),
				})
			case "completed", "cancelled":
				_, err = db.Exec(`UPDATE workspace_projects SET status=? WHERE workspace_id='workspace-1' AND id='project-1'`, testCase.change)
			default:
				t.Fatalf("unknown live authority change %q", testCase.change)
			}
			if err != nil {
				t.Fatal(err)
			}

			before := projectRequirementMutationEffectSnapshot(t, db)
			_, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
				WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 2,
				Content: projectRequirementTestContent("Must remain denied"), ChangeSummary: "Must remain denied",
				Actor: actor, OccurredAt: now.Add(4 * time.Minute),
			})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("SaveProjectRequirement(after %s) error = %v, want %v", testCase.change, err, testCase.wantErr)
			}
			after := projectRequirementMutationEffectSnapshot(t, db)
			if before != after {
				t.Fatalf("denied mutation after %s changed persisted effects\nbefore: %s\n after: %s", testCase.change, before, after)
			}
		})
	}
}

func TestProjectRequirementRepositoryCommitsGovernedLifecycleAndMaterialRereview(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 10, 10, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	created, err := repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
		ExpectedRevision: 0, Content: projectRequirementTestContent("Initial"), ChangeSummary: "Initial",
		IdempotencyKey: "lifecycle-key", RequestHash: strings.Repeat("d", 64), Actor: lead, OccurredAt: now,
	})
	if err != nil || created.Baseline == nil {
		t.Fatalf("SaveProjectRequirement(create) = %#v, %v", created.Baseline, err)
	}

	transition := func(action string, expected int64, actor contract.WorkspaceActor) contract.ProjectRequirementBaselineResponse {
		t.Helper()
		result, transitionErr := repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
			WorkspaceID: "workspace-1", ProjectID: "project-1", Action: action,
			ExpectedRevision: expected, Actor: actor, OccurredAt: now.Add(time.Duration(expected) * time.Minute),
		})
		if transitionErr != nil {
			t.Fatalf("TransitionProjectRequirement(%s) error = %v", action, transitionErr)
		}
		return result
	}
	if result := transition("submit_review", 1, lead); result.Baseline == nil || result.Baseline.Status != "in_review" || result.Baseline.CurrentRevision != 2 {
		t.Fatalf("submit result = %#v", result.Baseline)
	}
	if result := transition("approve", 2, owner); result.Baseline == nil || result.Baseline.Status != "approved" || result.Baseline.CurrentRevision != 3 {
		t.Fatalf("approve result = %#v", result.Baseline)
	}
	if result := transition("freeze", 3, owner); result.Baseline == nil || result.Baseline.Status != "frozen" || result.Baseline.EffectiveRevision == nil || *result.Baseline.EffectiveRevision != 4 {
		t.Fatalf("freeze result = %#v", result.Baseline)
	}
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 4,
		Content: projectRequirementTestContent("Plain frozen edit"), ChangeSummary: "Plain edit",
		Actor: lead, OccurredAt: now.Add(4 * time.Minute),
	}); !errors.Is(err, application.ErrProjectRequirementTransition) {
		t.Fatalf("SaveProjectRequirement(frozen plain) error = %v", err)
	}
	changed, err := repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 4,
		Content: projectRequirementTestContent("Materially changed"), ChangeSummary: "Material change", MaterialChange: true,
		Actor: lead, OccurredAt: now.Add(5 * time.Minute),
	})
	if err != nil || changed.Baseline == nil || changed.Baseline.Status != "changed" || changed.Baseline.CurrentRevision != 5 || changed.Baseline.EffectiveRevision == nil || *changed.Baseline.EffectiveRevision != 4 {
		t.Fatalf("SaveProjectRequirement(material) = %#v, %v", changed.Baseline, err)
	}
	if _, err = repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Action: "submit_review", ExpectedRevision: 4,
		Actor: lead, OccurredAt: now.Add(6 * time.Minute),
	}); !errors.Is(err, contract.ErrRevisionConflict) {
		t.Fatalf("TransitionProjectRequirement(stale) error = %v", err)
	}
	transition("submit_review", 5, lead)
	transition("approve", 6, owner)
	refrozen := transition("freeze", 7, owner)
	if refrozen.Baseline == nil || refrozen.Baseline.EffectiveRevision == nil || *refrozen.Baseline.EffectiveRevision != 8 {
		t.Fatalf("refrozen result = %#v", refrozen.Baseline)
	}
	retired := transition("retire", 8, owner)
	if retired.Baseline == nil || retired.Baseline.Status != "retired" || retired.Baseline.CurrentRevision != 9 {
		t.Fatalf("retire result = %#v", retired.Baseline)
	}
	if _, err = db.Exec(`UPDATE workspace_projects SET status='completed' WHERE id='project-1'`); err != nil {
		t.Fatal(err)
	}
	read, err := repository.ReadProjectRequirement(context.Background(), "workspace-1", "project-1", lead)
	if err != nil {
		t.Fatalf("ReadProjectRequirement(completed project) error = %v", err)
	}
	if read.Baseline == nil || read.Baseline.Status != "retired" || len(read.History) != 9 {
		t.Fatalf("completed-project read = baseline %#v, history %d", read.Baseline, len(read.History))
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_revisions", 9)
	assertProjectRequirementRowCount(t, db, "workspace_audit_entries", 9)
	assertProjectRequirementRowCount(t, db, "workspace_outbox_events", 9)
}

func TestProjectRequirementRepositoryRejectsSelfApprovalWithoutPartialEffects(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	now := time.Date(2026, 8, 19, 10, 30, 0, 0, time.UTC)
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Content: projectRequirementTestContent("Owner-authored"), ChangeSummary: "Owner-authored",
		IdempotencyKey: "self-key", RequestHash: strings.Repeat("e", 64), Actor: owner, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Action: "submit_review", ExpectedRevision: 1,
		Actor: owner, OccurredAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Action: "approve", ExpectedRevision: 2,
		Actor: owner, OccurredAt: now.Add(2 * time.Minute),
	}); !errors.Is(err, application.ErrProjectRequirementSelfApproval) {
		t.Fatalf("TransitionProjectRequirement(self approve) error = %v", err)
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_revisions", 2)
	assertProjectRequirementRowCount(t, db, "workspace_audit_entries", 2)
	assertProjectRequirementRowCount(t, db, "workspace_outbox_events", 2)
}

func TestProjectRequirementRepositoryLinksSameProjectIssueAndProjectsMaterialImpactWithoutIssueMutation(t *testing.T) {
	db := openProjectRequirementDB(t)
	if _, err := db.Exec(`INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,description,status,priority,creator_type,creator_id,
		project_id,metadata,properties,asset_ids,created_at,updated_at
	) VALUES('issue-1','workspace-1',1,'ONE-1','Original title','Original body','todo','high','member','lead-1',
		'project-1','{"safe":"metadata"}','{"safe":"properties"}','["asset-1"]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Content: projectRequirementTestContent("Initial"), ChangeSummary: "Initial", IdempotencyKey: "link-key",
		RequestHash: strings.Repeat("f", 64), Actor: lead, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	linked, err := repository.MutateProjectRequirementLink(context.Background(), application.ProjectRequirementLinkMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RequirementKey: "goal-1", TargetKind: "issue",
		TargetID: "issue-1", ExpectedRevision: 1, Actor: lead, OccurredAt: now.Add(time.Minute),
	})
	if err != nil || linked.Baseline == nil || linked.Baseline.CurrentRevision != 2 {
		t.Fatalf("MutateProjectRequirementLink(issue) = %#v, %v", linked.Baseline, err)
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_issue_links", 1)

	transition := func(action string, expected int64, actor contract.WorkspaceActor) {
		t.Helper()
		if _, transitionErr := repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
			WorkspaceID: "workspace-1", ProjectID: "project-1", Action: action,
			ExpectedRevision: expected, Actor: actor, OccurredAt: now.Add(time.Duration(expected) * time.Minute),
		}); transitionErr != nil {
			t.Fatalf("TransitionProjectRequirement(%s) error = %v", action, transitionErr)
		}
	}
	transition("submit_review", 2, lead)
	transition("approve", 3, owner)
	transition("freeze", 4, owner)

	var beforeIssue string
	if err = db.QueryRow(`SELECT json_array(id,workspace_id,number,identifier,title,description,status,priority,
		creator_type,creator_id,project_id,metadata,properties,asset_ids,created_at,updated_at)
		FROM workspace_issues WHERE id='issue-1'`).Scan(&beforeIssue); err != nil {
		t.Fatal(err)
	}
	changed, err := repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 5,
		Content: projectRequirementTestContent("Material impact"), ChangeSummary: "Material impact", MaterialChange: true,
		Actor: lead, OccurredAt: now.Add(6 * time.Minute),
	})
	if err != nil || changed.Baseline == nil || changed.Baseline.Status != "changed" || changed.Baseline.CurrentRevision != 6 {
		t.Fatalf("SaveProjectRequirement(material impact) = %#v, %v", changed.Baseline, err)
	}
	var afterIssue string
	if err = db.QueryRow(`SELECT json_array(id,workspace_id,number,identifier,title,description,status,priority,
		creator_type,creator_id,project_id,metadata,properties,asset_ids,created_at,updated_at)
		FROM workspace_issues WHERE id='issue-1'`).Scan(&afterIssue); err != nil {
		t.Fatal(err)
	}
	if afterIssue != beforeIssue {
		t.Fatalf("material Requirement change mutated Issue\nbefore: %s\n after: %s", beforeIssue, afterIssue)
	}
	var key, issueID, status string
	var sourceRevision int64
	if err = db.QueryRow(`SELECT requirement_key,issue_id,source_revision,status
		FROM workspace_requirement_review_projections WHERE baseline_id='baseline-1'`).Scan(&key, &issueID, &sourceRevision, &status); err != nil {
		t.Fatal(err)
	}
	if key != "goal-1" || issueID != "issue-1" || sourceRevision != 6 || status != "review_required" {
		t.Fatalf("review projection = key %q issue %q revision %d status %q", key, issueID, sourceRevision, status)
	}
	read, err := repository.ReadProjectRequirement(context.Background(), "workspace-1", "project-1", lead)
	if err != nil {
		t.Fatalf("ReadProjectRequirement(material impact) error = %v", err)
	}
	if read.CurrentContent == nil || read.CurrentContent.ProblemStatement != "Material impact" {
		t.Fatalf("current content = %#v", read.CurrentContent)
	}
	if read.EffectiveContent == nil || read.EffectiveContent.ProblemStatement != "Initial" {
		t.Fatalf("effective content = %#v", read.EffectiveContent)
	}
	if len(read.History) != 6 || read.History[0].Revision != 1 || read.History[5].Revision != 6 {
		t.Fatalf("history = %#v", read.History)
	}
	if len(read.IssueLinks) != 1 || read.IssueLinks[0].IssueID != "issue-1" || read.IssueLinks[0].Identifier != "ONE-1" || !read.IssueLinks[0].ReviewRequired {
		t.Fatalf("issue links = %#v", read.IssueLinks)
	}
}

func TestProjectRequirementRepositoryRejectsForeignOrNonTraceableIssueLinkWithoutRevisionAdvance(t *testing.T) {
	db := openProjectRequirementDB(t)
	for _, statement := range []string{
		`INSERT INTO workspace_projects(id,workspace_id,name,status,created_at,updated_at) VALUES('project-2','workspace-1','Foreign','planned','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at)
			VALUES('issue-2','workspace-1',2,'ONE-2','Foreign','todo','none','member','lead-1','project-2','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	now := time.Date(2026, 8, 19, 11, 30, 0, 0, time.UTC)
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Content: projectRequirementTestContent("Initial"), ChangeSummary: "Initial", IdempotencyKey: "reject-link-key",
		RequestHash: strings.Repeat("1", 64), Actor: lead, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []application.ProjectRequirementLinkMutation{
		{WorkspaceID: "workspace-1", ProjectID: "project-1", RequirementKey: "missing-key", TargetKind: "issue", TargetID: "issue-2", ExpectedRevision: 1, Actor: lead, OccurredAt: now},
		{WorkspaceID: "workspace-1", ProjectID: "project-1", RequirementKey: "goal-1", TargetKind: "issue", TargetID: "issue-2", ExpectedRevision: 1, Actor: lead, OccurredAt: now},
	} {
		if _, err = repository.MutateProjectRequirementLink(context.Background(), mutation); !errors.Is(err, application.ErrInvalidProjectRequirementRequest) && !errors.Is(err, application.ErrProjectRequirementNotFound) {
			t.Fatalf("MutateProjectRequirementLink(%+v) error = %v", mutation, err)
		}
	}
	var revision int64
	if err = db.QueryRow(`SELECT current_revision FROM workspace_requirement_baselines WHERE id='baseline-1'`).Scan(&revision); err != nil || revision != 1 {
		t.Fatalf("baseline revision after denied links = %d, %v", revision, err)
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_issue_links", 0)
	assertProjectRequirementRowCount(t, db, "workspace_requirement_revisions", 1)
}

func TestProjectRequirementRepositoryGrantsEditorAndPersistsStableRootOutlineNode(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
	ordinary := contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}
	access, err := repository.ReplaceProjectRequirementAccess(context.Background(), application.ProjectRequirementAccessReplace{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Grants: []application.ProjectRequirementGrantChange{
			{MemberID: "ordinary-member", GrantKind: "project_editor"},
			{MemberID: "ordinary-member", GrantKind: "requirement_approver"},
		},
		Actor: owner, OccurredAt: now,
	})
	if err != nil || access.Revision != 1 || len(access.Grants) != 2 {
		t.Fatalf("ReplaceProjectRequirementAccess() = %#v, %v", access, err)
	}
	if _, err = repository.ReplaceProjectRequirementAccess(context.Background(), application.ProjectRequirementAccessReplace{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Grants: nil, Actor: owner, OccurredAt: now.Add(time.Minute),
	}); !errors.Is(err, contract.ErrRevisionConflict) {
		t.Fatalf("ReplaceProjectRequirementAccess(stale) error = %v", err)
	}

	outlineCommand := application.ProjectOutlineNodeCreate{
		NodeID: "outline-1", WorkspaceID: "workspace-1", ProjectID: "project-1",
		ExpectedRevision: 0, Title: "Root delivery scope", IdempotencyKey: "outline-key",
		RequestHash: strings.Repeat("2", 64), Actor: ordinary, OccurredAt: now.Add(2 * time.Minute),
	}
	outline, err := repository.CreateProjectOutlineNode(context.Background(), outlineCommand)
	if err != nil || outline.Revision != 1 || len(outline.Nodes) != 1 || outline.Nodes[0].ID != "outline-1" {
		t.Fatalf("CreateProjectOutlineNode() = %#v, %v", outline, err)
	}
	replayCommand := outlineCommand
	replayCommand.NodeID = "ignored-retry-node"
	replayed, err := repository.CreateProjectOutlineNode(context.Background(), replayCommand)
	if err != nil || len(replayed.Nodes) != 1 || replayed.Nodes[0].ID != "outline-1" {
		t.Fatalf("CreateProjectOutlineNode(replay) = %#v, %v", replayed, err)
	}

	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Content: projectRequirementTestContent("Editor baseline"), ChangeSummary: "Editor baseline",
		IdempotencyKey: "editor-baseline-key", RequestHash: strings.Repeat("3", 64), Actor: ordinary, OccurredAt: now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	linked, err := repository.MutateProjectRequirementLink(context.Background(), application.ProjectRequirementLinkMutation{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RequirementKey: "goal-1", TargetKind: "outline",
		TargetID: "outline-1", ExpectedRevision: 1, Actor: ordinary, OccurredAt: now.Add(4 * time.Minute),
	})
	if err != nil || linked.Baseline == nil || linked.Baseline.CurrentRevision != 2 {
		t.Fatalf("MutateProjectRequirementLink(outline) = %#v, %v", linked.Baseline, err)
	}
	readRequirement, err := repository.ReadProjectRequirement(context.Background(), "workspace-1", "project-1", ordinary)
	if err != nil {
		t.Fatalf("ReadProjectRequirement(outline link) error = %v", err)
	}
	if len(readRequirement.OutlineLinks) != 1 || readRequirement.OutlineLinks[0].NodeID != "outline-1" || readRequirement.OutlineLinks[0].NodeTitle != "Root delivery scope" {
		t.Fatalf("outline links = %#v", readRequirement.OutlineLinks)
	}
	assertProjectRequirementRowCount(t, db, "workspace_project_outline_nodes", 1)
	assertProjectRequirementRowCount(t, db, "workspace_requirement_outline_links", 1)

	read, err := repository.ReadProjectOutline(context.Background(), "workspace-1", "project-1", ordinary)
	if err != nil || read.Revision != 1 || len(read.Nodes) != 1 || read.Nodes[0].ID != "outline-1" || read.Nodes[0].Title != "Root delivery scope" {
		t.Fatalf("ReadProjectOutline() = %#v, %v", read, err)
	}
}

func TestProjectRequirementRepositoryDoesNotTreatProjectLeadAsOutlineEditor(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	_, err = repository.CreateProjectOutlineNode(context.Background(), application.ProjectOutlineNodeCreate{
		NodeID: "outline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Title: "Denied root", IdempotencyKey: "denied-outline-key", RequestHash: strings.Repeat("4", 64),
		Actor: lead, OccurredAt: time.Date(2026, 8, 19, 12, 30, 0, 0, time.UTC),
	})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("CreateProjectOutlineNode(project lead) error = %v", err)
	}
	assertProjectRequirementRowCount(t, db, "workspace_project_outline_sets", 0)
	assertProjectRequirementRowCount(t, db, "workspace_project_outline_nodes", 0)
	if _, err = repository.ReplaceProjectRequirementAccess(context.Background(), application.ProjectRequirementAccessReplace{
		WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Actor: lead, OccurredAt: time.Date(2026, 8, 19, 12, 31, 0, 0, time.UTC),
	}); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("ReplaceProjectRequirementAccess(project lead) error = %v", err)
	}
}

func TestProjectRequirementRepositoryRejectsCorruptStoredLifecycleTimestamp(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 12, 40, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Content: projectRequirementTestContent("Initial"), ChangeSummary: "Initial", IdempotencyKey: "corrupt-time-key",
		RequestHash: strings.Repeat("5", 64), Actor: lead, OccurredAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Action: "submit_review", ExpectedRevision: 1,
		Actor: lead, OccurredAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE workspace_requirement_baselines SET submitted_at='not-a-time' WHERE id='baseline-1'`); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ReadProjectRequirement(context.Background(), "workspace-1", "project-1", lead); err == nil {
		t.Fatal("ReadProjectRequirement(corrupt timestamp) error = nil, want safe failure")
	}
}

func TestProjectRequirementRepositoryRollsBackEveryCreateEffectWhenOutboxFails(t *testing.T) {
	db := openProjectRequirementDB(t)
	if _, err := db.Exec(`CREATE TRIGGER fail_project_requirement_outbox
		BEFORE INSERT ON workspace_outbox_events
		WHEN NEW.aggregate_kind='requirement_baseline'
		BEGIN SELECT RAISE(ABORT, 'injected Project Requirement outbox failure'); END`); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
		BaselineID: "baseline-rollback", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
		Content: projectRequirementTestContent("Must roll back"), ChangeSummary: "Must roll back",
		IdempotencyKey: "rollback-key", RequestHash: strings.Repeat("6", 64),
		Actor: contract.WorkspaceActor{Type: "member", ID: "owner-1"}, OccurredAt: time.Date(2026, 8, 19, 12, 50, 0, 0, time.UTC),
	})
	if err == nil || strings.Contains(err.Error(), "Must roll back") {
		t.Fatalf("SaveProjectRequirement(injected failure) error = %v", err)
	}
	for _, table := range []string{
		"workspace_requirement_baselines", "workspace_requirement_revisions", "workspace_mutation_idempotency",
		"workspace_audit_entries", "workspace_outbox_events",
	} {
		assertProjectRequirementRowCount(t, db, table, 0)
	}
}

func TestProjectRequirementRepositoryAllowsOneConcurrentInitialCreateWinner(t *testing.T) {
	db := openProjectRequirementDB(t)
	db.SetMaxOpenConns(4)
	repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errorsByCommand := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsByCommand {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsByCommand[index] = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
				BaselineID: "baseline-concurrent-" + string(rune('a'+index)), WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
				Content: projectRequirementTestContent("Concurrent"), ChangeSummary: "Concurrent create",
				IdempotencyKey: "concurrent-key-" + string(rune('a'+index)), RequestHash: strings.Repeat(string(rune('7'+index)), 64),
				Actor: contract.WorkspaceActor{Type: "member", ID: "owner-1"}, OccurredAt: time.Date(2026, 8, 19, 13, index, 0, 0, time.UTC),
			})
		}(index)
	}
	close(start)
	wait.Wait()
	winners, conflicts := 0, 0
	for _, commandErr := range errorsByCommand {
		switch {
		case commandErr == nil:
			winners++
		case errors.Is(commandErr, contract.ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("concurrent create error = %v", commandErr)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("concurrent create outcomes = winners %d conflicts %d errors %#v", winners, conflicts, errorsByCommand)
	}
	assertProjectRequirementRowCount(t, db, "workspace_requirement_baselines", 1)
	assertProjectRequirementRowCount(t, db, "workspace_requirement_revisions", 1)
	assertProjectRequirementRowCount(t, db, "workspace_audit_entries", 1)
	assertProjectRequirementRowCount(t, db, "workspace_outbox_events", 1)
	assertProjectRequirementRowCount(t, db, "workspace_mutation_idempotency", 1)
}

func TestUserAndBatchIssueDeletionCloseCanonicalLinksWithoutLegacyWrites(t *testing.T) {
	for _, batch := range []bool{false, true} {
		name := "user Issue deletion"
		if batch {
			name = "batch Issue deletion"
		}
		t.Run(name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			if _, err := db.Exec(`INSERT INTO workspace_issues(
				id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,project_id,created_at,updated_at
			) VALUES('issue-1','workspace-1',1,'ONE-1','Linked Issue','todo','none','member','lead-1','project-1','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO workspace_requirements(
				id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at
			) VALUES('legacy-1','workspace-1','project-1','Legacy',1,'draft','covered','["issue-1"]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at)
				VALUES('legacy-v1','legacy-1',1,'Legacy content','2026-08-19T00:00:00Z')`); err != nil {
				t.Fatal(err)
			}
			repository, err := persistence.NewProjectRequirementRepository(persistence.Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 8, 19, 13, 30, 0, 0, time.UTC)
			lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
			owner := contract.WorkspaceActor{Type: "member", ID: "owner-1"}
			if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
				BaselineID: "baseline-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 0,
				Content: projectRequirementTestContent("Before deletion"), ChangeSummary: "Initial", IdempotencyKey: "delete-baseline-key",
				RequestHash: strings.Repeat("a", 64), Actor: lead, OccurredAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			if _, err = repository.MutateProjectRequirementLink(context.Background(), application.ProjectRequirementLinkMutation{
				WorkspaceID: "workspace-1", ProjectID: "project-1", RequirementKey: "goal-1", TargetKind: "issue",
				TargetID: "issue-1", ExpectedRevision: 1, Actor: lead, OccurredAt: now.Add(time.Minute),
			}); err != nil {
				t.Fatal(err)
			}
			for _, transition := range []struct {
				action   string
				expected int64
				actor    contract.WorkspaceActor
			}{
				{action: "submit_review", expected: 2, actor: lead},
				{action: "approve", expected: 3, actor: owner},
				{action: "freeze", expected: 4, actor: owner},
			} {
				if _, err = repository.TransitionProjectRequirement(context.Background(), application.ProjectRequirementTransition{
					WorkspaceID: "workspace-1", ProjectID: "project-1", Action: transition.action,
					ExpectedRevision: transition.expected, Actor: transition.actor, OccurredAt: now.Add(time.Duration(transition.expected) * time.Minute),
				}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = repository.SaveProjectRequirement(context.Background(), application.ProjectRequirementSave{
				WorkspaceID: "workspace-1", ProjectID: "project-1", ExpectedRevision: 5,
				Content: projectRequirementTestContent("Material change"), ChangeSummary: "Material change", MaterialChange: true,
				Actor: lead, OccurredAt: now.Add(6 * time.Minute),
			}); err != nil {
				t.Fatal(err)
			}

			if batch {
				issues, createErr := persistence.NewIssueRepository(persistence.Config{DB: db})
				if createErr != nil {
					t.Fatal(createErr)
				}
				batchIssues, ok := issues.(interface {
					BatchDelete(context.Context, application.IssueBatchDeleteCommand) ([]string, error)
				})
				if !ok {
					t.Fatal("SQLite Issue repository does not expose batch deletion")
				}
				if _, err = batchIssues.BatchDelete(context.Background(), application.IssueBatchDeleteCommand{
					WorkspaceID: "workspace-1", IssueIDs: []string{"ONE-1"}, Now: now.Add(7 * time.Minute),
				}); err != nil {
					t.Fatal(err)
				}
			} else {
				deletion, createErr := persistence.NewIssueDeletionRepository(persistence.Config{DB: db})
				if createErr != nil {
					t.Fatal(createErr)
				}
				if _, err = deletion.Delete(context.Background(), "workspace-1", "ONE-1"); err != nil {
					t.Fatal(err)
				}
			}

			var currentRevision int64
			if err = db.QueryRow(`SELECT current_revision FROM workspace_requirement_baselines WHERE id='baseline-1'`).Scan(&currentRevision); err != nil || currentRevision != 7 {
				t.Fatalf("baseline revision after Issue deletion = %d, %v", currentRevision, err)
			}
			var action, actorID string
			if err = db.QueryRow(`SELECT action,actor_id FROM workspace_requirement_revisions WHERE baseline_id='baseline-1' AND revision=7`).Scan(&action, &actorID); err != nil || action != "issue_deleted" || actorID != "system:issue-deletion" {
				t.Fatalf("Issue deletion revision = action %q actor %q error %v", action, actorID, err)
			}
			var unlinkedRevision sql.NullInt64
			var unlinkedBy sql.NullString
			if err = db.QueryRow(`SELECT unlinked_revision,unlinked_by FROM workspace_requirement_issue_links WHERE baseline_id='baseline-1' AND issue_id='issue-1'`).Scan(&unlinkedRevision, &unlinkedBy); err != nil || !unlinkedRevision.Valid || unlinkedRevision.Int64 != 7 || !unlinkedBy.Valid || unlinkedBy.String != "system:issue-deletion" {
				t.Fatalf("closed Issue link = revision %+v actor %+v error %v", unlinkedRevision, unlinkedBy, err)
			}
			assertProjectRequirementRowCount(t, db, "workspace_requirement_review_projections", 0)
			var legacyVersion int
			var legacyIssueIDs string
			if err = db.QueryRow(`SELECT current_version,issue_ids FROM workspace_requirements WHERE id='legacy-1'`).Scan(&legacyVersion, &legacyIssueIDs); err != nil || legacyVersion != 1 || legacyIssueIDs != `["issue-1"]` {
				t.Fatalf("legacy Requirement mutated = version %d issues %q error %v", legacyVersion, legacyIssueIDs, err)
			}
			var legacyVersions int
			if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id='legacy-1'`).Scan(&legacyVersions); err != nil || legacyVersions != 1 {
				t.Fatalf("legacy Requirement versions = %d, %v", legacyVersions, err)
			}
		})
	}
}

func projectRequirementTestContent(problem string) requirementDomain.Content {
	return requirementDomain.Content{
		ProblemStatement: problem,
		Goals:            []requirementDomain.Item{{Key: "goal-1", Text: "Deliver"}},
	}
}

func projectRequirementMutationEffectSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	queries := []string{
		`SELECT json_array(id,workspace_id,project_id,status,current_revision,approved_revision,effective_revision,
			review_origin,latest_content_author,submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,
			retired_by,retired_at,legacy_requirement_id,legacy_snapshot_json,created_at,updated_at)
			FROM workspace_requirement_baselines WHERE id='baseline-1'`,
		`SELECT json_array(baseline_id,revision,content_json,status,action,change_summary,actor_id,submitted_by,
			submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at)
			FROM workspace_requirement_revisions WHERE baseline_id='baseline-1' ORDER BY revision DESC LIMIT 1`,
		`SELECT json_array(
			(SELECT COUNT(*) FROM workspace_requirement_baselines),
			(SELECT COUNT(*) FROM workspace_requirement_revisions),
			(SELECT COUNT(*) FROM workspace_requirement_issue_links),
			(SELECT COUNT(*) FROM workspace_requirement_outline_links),
			(SELECT COUNT(*) FROM workspace_requirement_review_projections),
			(SELECT COUNT(*) FROM workspace_project_requirement_access_sets),
			(SELECT COUNT(*) FROM workspace_project_requirement_grants),
			(SELECT COUNT(*) FROM workspace_project_outline_sets),
			(SELECT COUNT(*) FROM workspace_project_outline_nodes),
			(SELECT COUNT(*) FROM workspace_mutation_idempotency WHERE resource_kind IN ('requirement_baseline','project_requirement_access','project_outline')),
			(SELECT COUNT(*) FROM workspace_resource_revisions WHERE resource_kind IN ('requirement_baseline','project_requirement_access','project_outline')),
			(SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind IN ('requirement_baseline','project_requirement_access','project_outline')),
			(SELECT COUNT(*) FROM workspace_outbox_events WHERE aggregate_kind IN ('requirement_baseline','project_requirement_access','project_outline'))
		)`,
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

func openProjectRequirementDB(t *testing.T) *sql.DB {
	return openProjectRequirementDBWithDriver(t, "sqlite")
}

func openProjectRequirementDBWithDriver(t *testing.T, driverName string) *sql.DB {
	return openProjectRequirementDBAtPath(t, driverName, filepath.Join(t.TempDir(), "project-requirements.db"))
}

func openProjectRequirementDBAtPath(t *testing.T, driverName, databasePath string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
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
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,issue_prefix,created_at,updated_at) VALUES
			('workspace-1','Workspace One','workspace-one','ONE','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,lead_type,lead_id,asset_ids,created_at,updated_at) VALUES
			('project-1','workspace-1','Project One','in_progress','none','member','lead-member','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('owner-member','workspace-1','owner-1','owner','2026-08-19T00:00:00Z'),
			('lead-member','workspace-1','lead-1','member','2026-08-19T00:00:00Z'),
			('ordinary-member','workspace-1','ordinary-1','member','2026-08-19T00:00:00Z')`,
	} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func marshalProjectRequirementTestContent(t *testing.T, content requirementDomain.Content) string {
	t.Helper()
	encoded, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

var projectRequirementCountingDriverSequence atomic.Uint64

type projectRequirementSQLQueryCounter struct {
	value atomic.Int64
}

func (c *projectRequirementSQLQueryCounter) Reset()      { c.value.Store(0) }
func (c *projectRequirementSQLQueryCounter) Load() int64 { return c.value.Load() }

type projectRequirementCountingDriver struct {
	delegate driver.Driver
	counter  *projectRequirementSQLQueryCounter
}

func (d *projectRequirementCountingDriver) Open(name string) (driver.Conn, error) {
	connection, err := d.delegate.Open(name)
	if err != nil {
		return nil, err
	}
	return &projectRequirementCountingConnection{Conn: connection, counter: d.counter}, nil
}

type projectRequirementCountingConnection struct {
	driver.Conn
	counter *projectRequirementSQLQueryCounter
}

func (c *projectRequirementCountingConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	c.counter.value.Add(1)
	return queryer.QueryContext(ctx, query, args)
}

func (c *projectRequirementCountingConnection) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.ExecContext(ctx, query, args)
}

func assertProjectRequirementRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
