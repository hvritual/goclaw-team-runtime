package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

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
	if _, err := repository.SaveProjectRequirement(context.Background(), conflictCommand); !errors.Is(err, application.ErrProjectRequirementConflict) {
		t.Fatalf("SaveProjectRequirement(conflict) error = %v, want ErrProjectRequirementConflict", err)
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

func projectRequirementTestContent(problem string) requirementDomain.Content {
	return requirementDomain.Content{
		ProblemStatement: problem,
		Goals:            []requirementDomain.Item{{Key: "goal-1", Text: "Deliver"}},
	}
}

func openProjectRequirementDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "project-requirements.db"))
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
