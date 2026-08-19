package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

func TestProjectRetrospectiveRepositoryCreatesGovernedDraftAndReplays(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 14, 0, 0, 0, time.UTC)
	command := application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-1",
		Content: retrospectiveDomain.Content{
			Summary: "Release learning", Lessons: []string{"Review earlier"},
			ActionItems: []retrospectiveDomain.ActionItem{{ID: "action-1", Title: "Schedule review"}},
		},
		Participants:   []retrospectiveDomain.Participant{{MemberID: "ordinary-member", Role: retrospectiveDomain.RoleParticipant}},
		IdempotencyKey: "create-retro-1", RequestHash: strings.Repeat("a", 64),
		Actor: contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}, OccurredAt: now,
	}
	created, err := repository.CreateProjectRetrospective(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "retro-1" || created.Status != retrospectiveDomain.StatusDraft || created.CurrentRevision != 1 || created.Current == nil || created.Current.Content.Summary != "Release learning" {
		t.Fatalf("created = %#v", created)
	}
	if len(created.Current.Participants) != 1 || created.Current.Participants[0].MemberID != "ordinary-member" {
		t.Fatalf("participants = %#v", created.Current.Participants)
	}

	replayCommand := command
	replayCommand.RetrospectiveID = "must-not-be-created"
	replayed, err := repository.CreateProjectRetrospective(context.Background(), replayCommand)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != created.ID || replayed.CurrentRevision != created.CurrentRevision {
		t.Fatalf("replayed = %#v, created = %#v", replayed, created)
	}
	conflict := command
	conflict.RequestHash = strings.Repeat("b", 64)
	if _, err = repository.CreateProjectRetrospective(context.Background(), conflict); !errors.Is(err, contract.ErrIdempotencyConflict) {
		t.Fatalf("conflict error = %v, want %v", err, contract.ErrIdempotencyConflict)
	}

	for table, want := range map[string]int{
		"workspace_project_retrospectives":             1,
		"workspace_project_retrospective_revisions":    1,
		"workspace_project_retrospective_participants": 1,
		"workspace_mutation_idempotency":               1,
		"workspace_audit_entries":                      1,
		"workspace_outbox_events":                      1,
	} {
		var count int
		if err = db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, want %d, error %v", table, count, want, err)
		}
	}
	var auditMetadata, outboxPayload string
	if err = db.QueryRow(`SELECT metadata_json FROM workspace_audit_entries WHERE action='workspace.project.retrospective.create'`).Scan(&auditMetadata); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT payload_json FROM workspace_outbox_events WHERE event_type='retrospective:drafted'`).Scan(&outboxPayload); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"Release learning", "Review earlier", "Schedule review"} {
		if strings.Contains(auditMetadata, forbidden) || strings.Contains(outboxPayload, forbidden) {
			t.Fatalf("content leaked into governance evidence: %q", forbidden)
		}
	}
}

func TestProjectRetrospectiveRepositoryCreateReplayIsExactAfterLaterMutationAndReauthorizes(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 14, 30, 0, 0, time.UTC)
	command := application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-exact-replay",
		Content: retrospectiveDomain.Content{
			Summary: "Original response", Lessons: []string{"Keep exact"},
			ActionItems: []retrospectiveDomain.ActionItem{{ID: "action-1", Title: "Follow through"}},
		},
		IdempotencyKey: "exact-replay", RequestHash: strings.Repeat("8", 64),
		Actor: contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}, OccurredAt: now,
	}
	created, err := repository.CreateProjectRetrospective(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, RequestID: "publish-before-replay",
		Actor: contract.WorkspaceActor{Type: "member", ID: "lead-1"}, OccurredAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	replayCommand := command
	replayCommand.RetrospectiveID = "ignored-on-replay"
	replayed, err := repository.CreateProjectRetrospective(context.Background(), replayCommand)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(replayed, created) {
		t.Fatalf("replayed response changed\ncreated: %#v\nreplayed: %#v", created, replayed)
	}
	if _, err = db.Exec(`DELETE FROM auth_members WHERE workspace_id='workspace-1' AND id='ordinary-member'`); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreateProjectRetrospective(context.Background(), replayCommand); !errors.Is(err, contract.ErrActorOutsideWorkspace) {
		t.Fatalf("removed actor replay error = %v", err)
	}
	assertProjectRetrospectiveEffectCounts(t, db, created.ID, 2)
}

func TestProjectRetrospectiveRepositoryDeniesSelfAppointedFacilitator(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-denied",
		Content:        retrospectiveDomain.Content{Summary: "Summary", Lessons: []string{"Lesson"}},
		Participants:   []retrospectiveDomain.Participant{{MemberID: "ordinary-member", Role: retrospectiveDomain.RoleFacilitator}},
		IdempotencyKey: "denied", RequestHash: strings.Repeat("d", 64),
		Actor: contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}, OccurredAt: time.Date(2026, 8, 19, 14, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("self-appointed facilitator error = %v", err)
	}
	for _, table := range []string{"workspace_project_retrospectives", "workspace_project_retrospective_revisions", "workspace_project_retrospective_participants", "workspace_mutation_idempotency", "workspace_audit_entries", "workspace_outbox_events"} {
		var count int
		if queryErr := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); queryErr != nil || count != 0 {
			t.Fatalf("%s count = %d, error %v", table, count, queryErr)
		}
	}
}

func TestProjectRetrospectiveRepositoryPublishesSupersedesAndArchives(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	facilitator := contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}
	participants := []retrospectiveDomain.Participant{
		{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant},
		{MemberID: "ordinary-member", Role: retrospectiveDomain.RoleFacilitator},
	}
	initialContent := retrospectiveDomain.Content{
		Summary: "Initial", Lessons: []string{"First lesson"},
		ActionItems: []retrospectiveDomain.ActionItem{{ID: "action-1", Title: "Follow up"}},
	}
	created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-lifecycle",
		Content: initialContent, Participants: participants, IdempotencyKey: "lifecycle-create", RequestHash: strings.Repeat("c", 64), Actor: lead, OccurredAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, Actor: facilitator, OccurredAt: now.Add(time.Minute), RequestID: "publish-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != retrospectiveDomain.StatusPublished || published.CurrentRevision != 2 || published.PublishedRevision == nil || *published.PublishedRevision != 2 {
		t.Fatalf("published = %#v", published)
	}

	changedContent := initialContent
	changedContent.Summary = "Second published snapshot"
	changedContent.Lessons = []string{"Second lesson"}
	superseding, err := repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 2, Action: retrospectiveDomain.ActionPublishRevision, Content: &changedContent, Participants: &participants,
		Actor: facilitator, OccurredAt: now.Add(2 * time.Minute), RequestID: "publish-3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if superseding.CurrentRevision != 3 || superseding.PublishedRevision == nil || *superseding.PublishedRevision != 3 || superseding.Current.Content.Summary != "Second published snapshot" {
		t.Fatalf("superseding = %#v", superseding)
	}
	var revisionTwoStatus, revisionTwoSummary string
	for _, revision := range superseding.History {
		if revision.Revision == 2 {
			revisionTwoStatus, revisionTwoSummary = revision.Status, revision.Content.Summary
		}
	}
	if revisionTwoStatus != "superseded" || revisionTwoSummary != "Initial" {
		t.Fatalf("revision 2 projection = status %q summary %q", revisionTwoStatus, revisionTwoSummary)
	}

	archived, err := repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 3, Action: retrospectiveDomain.ActionArchive, Actor: facilitator, OccurredAt: now.Add(3 * time.Minute), RequestID: "archive-4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if archived.Status != retrospectiveDomain.StatusArchived || archived.CurrentRevision != 4 || archived.PublishedRevision == nil || *archived.PublishedRevision != 3 {
		t.Fatalf("archived = %#v", archived)
	}
	var storedSecondSummary string
	if err = db.QueryRow(`SELECT json_extract(content_json,'$.summary') FROM workspace_project_retrospective_revisions
		WHERE workspace_id='workspace-1' AND project_id='project-1' AND retrospective_id='retro-lifecycle' AND revision=2`).Scan(&storedSecondSummary); err != nil || storedSecondSummary != "Initial" {
		t.Fatalf("stored published revision = %q, error %v", storedSecondSummary, err)
	}
	var resourceRevision, auditCount, outboxCount int
	if err = db.QueryRow(`SELECT revision FROM workspace_resource_revisions WHERE workspace_id='workspace-1' AND resource_kind='project_retrospective' AND resource_id='retro-lifecycle'`).Scan(&resourceRevision); err != nil || resourceRevision != 4 {
		t.Fatalf("resource revision = %d, error %v", resourceRevision, err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='project_retrospective' AND resource_id='retro-lifecycle'`).Scan(&auditCount); err != nil || auditCount != 4 {
		t.Fatalf("audit count = %d, error %v", auditCount, err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_outbox_events WHERE aggregate_kind='project_retrospective' AND aggregate_id='retro-lifecycle'`).Scan(&outboxCount); err != nil || outboxCount != 4 {
		t.Fatalf("outbox count = %d, error %v", outboxCount, err)
	}
}

func TestProjectRetrospectiveRepositoryProtectsLinkedPublishedActionItems(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 15, 30, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	participants := []retrospectiveDomain.Participant{{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant}}
	content := retrospectiveDomain.Content{
		Summary: "Published", Lessons: []string{"Keep provenance"},
		ActionItems: []retrospectiveDomain.ActionItem{{ID: "linked-action", Title: "Stable", Description: "Exact", AssigneeID: "ordinary-member", DueDate: "2026-08-30"}},
	}
	created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-linked",
		Content: content, Participants: participants, IdempotencyKey: "linked-create", RequestHash: strings.Repeat("9", 64), Actor: lead, OccurredAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, RequestID: "linked-publish", Actor: lead, OccurredAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO workspace_project_retrospective_action_links(
		workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,target_id,request_hash,claimed_by,claimed_at,linked_by,linked_at
	) VALUES('workspace-1','project-1','retro-linked','linked-action',2,'linked','task','task-1',?,'lead-1','2026-08-19T15:32:00Z','lead-1','2026-08-19T15:32:00Z')`, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	changed := content
	changed.ActionItems = append([]retrospectiveDomain.ActionItem(nil), content.ActionItems...)
	changed.ActionItems[0].Title = "Rewritten"
	_, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 2, Action: retrospectiveDomain.ActionPublishRevision, Content: &changed, Participants: &participants,
		RequestID: "linked-rewrite", Actor: lead, OccurredAt: now.Add(2 * time.Minute),
	})
	if !errors.Is(err, contract.ErrProjectRetrospectiveStateConflict) {
		t.Fatalf("linked rewrite error = %v", err)
	}
	assertProjectRetrospectiveEffectCounts(t, db, created.ID, 2)

	allowed := content
	allowed.Summary = "New retrospective summary"
	allowed.ActionItems = append([]retrospectiveDomain.ActionItem(nil), content.ActionItems...)
	allowed.ActionItems = append(allowed.ActionItems, retrospectiveDomain.ActionItem{ID: "free-action", Title: "New free item"})
	updated, err := repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 2, Action: retrospectiveDomain.ActionPublishRevision, Content: &allowed, Participants: &participants,
		RequestID: "linked-preserved", Actor: lead, OccurredAt: now.Add(3 * time.Minute),
	})
	if err != nil || updated.CurrentRevision != 3 || updated.Current.Content.ActionItems[0].Title != "Stable" {
		t.Fatalf("allowed published revision = %#v, error %v", updated, err)
	}
}

func TestProjectRetrospectiveRepositoryReauthorizesAndRejectsStaleWithoutEffects(t *testing.T) {
	now := time.Date(2026, 8, 19, 16, 0, 0, 0, time.UTC)
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	facilitator := contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}
	content := retrospectiveDomain.Content{Summary: "Summary", Lessons: []string{"Lesson"}}
	withFacilitator := []retrospectiveDomain.Participant{
		{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant},
		{MemberID: "ordinary-member", Role: retrospectiveDomain.RoleFacilitator},
	}

	t.Run("stale revision", func(t *testing.T) {
		db := openProjectRequirementDB(t)
		repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-stale", Content: content,
			Participants: withFacilitator, IdempotencyKey: "stale-create", RequestHash: strings.Repeat("e", 64), Actor: lead, OccurredAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
			ExpectedRevision: 2, Action: retrospectiveDomain.ActionPublish, Actor: facilitator, OccurredAt: now.Add(time.Minute), RequestID: "stale-publish",
		})
		var conflict contract.RevisionConflictError
		if !errors.As(err, &conflict) || conflict.CurrentRevision != 1 {
			t.Fatalf("stale error = %v", err)
		}
		assertProjectRetrospectiveEffectCounts(t, db, created.ID, 1)
	})

	t.Run("lead reassignment", func(t *testing.T) {
		db := openProjectRequirementDB(t)
		repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-reassigned", Content: content,
			Participants:   []retrospectiveDomain.Participant{{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant}},
			IdempotencyKey: "reassigned-create", RequestHash: strings.Repeat("f", 64), Actor: lead, OccurredAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`UPDATE workspace_projects SET lead_id='ordinary-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
			t.Fatal(err)
		}
		_, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
			ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, Actor: lead, OccurredAt: now.Add(time.Minute), RequestID: "reassigned-publish",
		})
		if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
			t.Fatalf("reassigned lead error = %v", err)
		}
		assertProjectRetrospectiveEffectCounts(t, db, created.ID, 1)
	})

	t.Run("facilitator removal", func(t *testing.T) {
		db := openProjectRequirementDB(t)
		repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-facilitator-removed", Content: content,
			Participants: withFacilitator, IdempotencyKey: "facilitator-create", RequestHash: strings.Repeat("1", 64), Actor: lead, OccurredAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		withoutFacilitator := []retrospectiveDomain.Participant{
			{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant},
			{MemberID: "ordinary-member", Role: retrospectiveDomain.RoleParticipant},
		}
		if _, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
			ExpectedRevision: 1, Action: retrospectiveDomain.ActionSaveDraft, Content: &content, Participants: &withoutFacilitator,
			Actor: lead, OccurredAt: now.Add(time.Minute), RequestID: "remove-facilitator",
		}); err != nil {
			t.Fatal(err)
		}
		_, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
			ExpectedRevision: 2, Action: retrospectiveDomain.ActionPublish, Actor: facilitator, OccurredAt: now.Add(2 * time.Minute), RequestID: "removed-facilitator-publish",
		})
		if !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
			t.Fatalf("removed facilitator error = %v", err)
		}
		assertProjectRetrospectiveEffectCounts(t, db, created.ID, 2)
	})

	t.Run("membership removal and foreign project", func(t *testing.T) {
		db := openProjectRequirementDB(t)
		repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-membership", Content: content,
			Participants: withFacilitator, IdempotencyKey: "membership-create", RequestHash: strings.Repeat("2", 64), Actor: lead, OccurredAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,status,priority,lead_type,lead_id,asset_ids,created_at,updated_at)
			VALUES('project-2','workspace-1','Project Two','in_progress','none','member','lead-member','[]','2026-08-19T00:00:00Z','2026-08-19T00:00:00Z')`); err != nil {
			t.Fatal(err)
		}
		_, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-2", RetrospectiveID: created.ID,
			ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, Actor: lead, OccurredAt: now.Add(time.Minute), RequestID: "foreign-project",
		})
		if !errors.Is(err, contract.ErrProjectRetrospectiveNotFound) {
			t.Fatalf("foreign project error = %v", err)
		}
		if _, err = db.Exec(`DELETE FROM auth_members WHERE workspace_id='workspace-1' AND id='ordinary-member'`); err != nil {
			t.Fatal(err)
		}
		_, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
			ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, Actor: facilitator, OccurredAt: now.Add(2 * time.Minute), RequestID: "removed-member",
		})
		if !errors.Is(err, contract.ErrActorOutsideWorkspace) {
			t.Fatalf("removed member error = %v", err)
		}
		assertProjectRetrospectiveEffectCounts(t, db, created.ID, 1)
	})
}

func assertProjectRetrospectiveEffectCounts(t *testing.T, db interface {
	QueryRow(string, ...any) *sql.Row
}, retrospectiveID string, want int) {
	t.Helper()
	for table, predicate := range map[string]string{
		"workspace_project_retrospective_revisions": "retrospective_id=?",
		"workspace_audit_entries":                   "resource_kind='project_retrospective' AND resource_id=?",
		"workspace_outbox_events":                   "aggregate_kind='project_retrospective' AND aggregate_id=?",
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE `+predicate, retrospectiveID).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count = %d, want %d, error %v", table, count, want, err)
		}
	}
}

func TestProjectRetrospectiveRepositoryReadsRestartedHistoryAndFailsClosed(t *testing.T) {
	t.Run("restart preserves immutable history and current access", func(t *testing.T) {
		databasePath := filepath.Join(t.TempDir(), "retrospective-restart.db")
		db := openProjectRequirementDBAtPath(t, "sqlite", databasePath)
		repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
		if err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 19, 17, 0, 0, 0, time.UTC)
		participants := []retrospectiveDomain.Participant{
			{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant},
			{MemberID: "ordinary-member", Role: retrospectiveDomain.RoleFacilitator},
		}
		created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-restart",
			Content: retrospectiveDomain.Content{Summary: "Persistent", Lessons: []string{"Restart"}}, Participants: participants,
			IdempotencyKey: "restart-create", RequestHash: strings.Repeat("3", 64), Actor: contract.WorkspaceActor{Type: "member", ID: "lead-1"}, OccurredAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
			ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, RequestID: "restart-publish",
			Actor: contract.WorkspaceActor{Type: "member", ID: "ordinary-1"}, OccurredAt: now.Add(time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		if err = db.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := sql.Open("sqlite", databasePath)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		repository, err = persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: reopened})
		if err != nil {
			t.Fatal(err)
		}
		read, err := repository.ReadProjectRetrospective(context.Background(), "workspace-1", "project-1", created.ID, contract.WorkspaceActor{Type: "member", ID: "ordinary-1"})
		if err != nil {
			t.Fatal(err)
		}
		if read.Status != retrospectiveDomain.StatusPublished || read.CurrentRevision != 2 || len(read.History) != 2 || read.Current == nil || read.Current.Content.Summary != "Persistent" {
			t.Fatalf("restarted read = %#v", read)
		}
		if read.Access.CanEdit || !read.Access.CanPublish || !read.Access.CanArchive {
			t.Fatalf("restarted access = %#v", read.Access)
		}
	})

	for _, testCase := range []struct {
		name    string
		corrupt string
	}{
		{name: "invalid stored content", corrupt: `UPDATE workspace_project_retrospective_revisions SET content_json='{"summary":"","lessons":[]}' WHERE retrospective_id='retro-corrupt'`},
		{name: "revision ownership drift", corrupt: `UPDATE workspace_project_retrospective_revisions SET project_id='project-drift' WHERE retrospective_id='retro-corrupt'`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := openProjectRequirementDB(t)
			repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			_, err = repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
				WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-corrupt",
				Content:        retrospectiveDomain.Content{Summary: "Valid", Lessons: []string{"Lesson"}},
				IdempotencyKey: "corrupt-create", RequestHash: strings.Repeat("4", 64), Actor: contract.WorkspaceActor{Type: "member", ID: "lead-1"}, OccurredAt: time.Date(2026, 8, 19, 17, 0, 0, 0, time.UTC),
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(`PRAGMA ignore_check_constraints=ON`); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(testCase.corrupt); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(`PRAGMA ignore_check_constraints=OFF`); err != nil {
				t.Fatal(err)
			}
			_, err = repository.ReadProjectRetrospective(context.Background(), "workspace-1", "project-1", "retro-corrupt", contract.WorkspaceActor{Type: "member", ID: "lead-1"})
			if !errors.Is(err, contract.ErrInvalidProjectRetrospective) {
				t.Fatalf("corrupt read error = %v", err)
			}
		})
	}
}
