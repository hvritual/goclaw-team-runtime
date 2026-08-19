package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	moderncSQLite "modernc.org/sqlite"
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

func TestProjectRetrospectiveRepositoryListsStablePagesAndExcludesArchivedByDefault(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	base := time.Date(2026, 8, 19, 20, 0, 0, 0, time.UTC)
	for index, id := range []string{"retro-a", "retro-b", "retro-c"} {
		created, createErr := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: id,
			Content:        retrospectiveDomain.Content{Summary: id, Lessons: []string{"Lesson"}},
			IdempotencyKey: "create-" + id, RequestHash: strings.Repeat(string(rune('a'+index)), 64), Actor: lead,
			OccurredAt: base.Add(time.Duration(min(index, 1)) * time.Minute),
		})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if id == "retro-c" {
			if _, err = repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
				WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
				ExpectedRevision: 1, Action: retrospectiveDomain.ActionArchive, RequestID: "archive-c", Actor: lead, OccurredAt: base.Add(2 * time.Minute),
			}); err != nil {
				t.Fatal(err)
			}
		}
	}
	first, err := repository.ListProjectRetrospectives(context.Background(), application.ProjectRetrospectiveListQuery{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Limit: 1, Actor: lead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Retrospectives) != 1 || first.Retrospectives[0].ID != "retro-b" || !first.HasMore {
		t.Fatalf("first page = %#v", first)
	}
	second, err := repository.ListProjectRetrospectives(context.Background(), application.ProjectRetrospectiveListQuery{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Limit: 1, Actor: lead,
		Cursor: &application.ProjectRetrospectiveListCursor{UpdatedAt: first.Retrospectives[0].UpdatedAt, ID: first.Retrospectives[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Retrospectives) != 1 || second.Retrospectives[0].ID != "retro-a" || second.HasMore {
		t.Fatalf("second page = %#v", second)
	}
	withArchived, err := repository.ListProjectRetrospectives(context.Background(), application.ProjectRetrospectiveListQuery{
		WorkspaceID: "workspace-1", ProjectID: "project-1", Limit: 3, IncludeArchived: true, Actor: lead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(withArchived.Retrospectives) != 3 || withArchived.Retrospectives[0].ID != "retro-c" || withArchived.Retrospectives[0].Status != retrospectiveDomain.StatusArchived {
		t.Fatalf("archived page = %#v", withArchived)
	}
	for _, value := range withArchived.Retrospectives {
		if value.Current == nil || len(value.History) == 0 || value.ActionLinks == nil {
			t.Fatalf("incomplete list projection = %#v", value)
		}
	}
}

func TestProjectRetrospectiveRepositoryListHasConstantQueryBound(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "project-retrospective-query-bound.db")
	seedDB := openProjectRequirementDBAtPath(t, "sqlite", databasePath)
	if err := seedDB.Close(); err != nil {
		t.Fatal(err)
	}
	counter := &projectRequirementSQLQueryCounter{}
	driverName := fmt.Sprintf("project-retrospective-counting-sqlite-%d", projectRequirementCountingDriverSequence.Add(1))
	sql.Register(driverName, &projectRequirementCountingDriver{delegate: &moderncSQLite.Driver{}, counter: counter})
	db, err := sql.Open(driverName, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	create := func(index int) {
		t.Helper()
		id := fmt.Sprintf("retro-bound-%02d", index)
		_, createErr := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
			WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: id,
			Content:        retrospectiveDomain.Content{Summary: id, Lessons: []string{"Lesson"}},
			IdempotencyKey: "create-" + id, RequestHash: fmt.Sprintf("%064x", index+1), Actor: lead,
			OccurredAt: time.Date(2026, 8, 19, 21, index, 0, 0, time.UTC),
		})
		if createErr != nil {
			t.Fatal(createErr)
		}
	}
	create(0)
	list := func() int64 {
		t.Helper()
		counter.Reset()
		page, listErr := repository.ListProjectRetrospectives(context.Background(), application.ProjectRetrospectiveListQuery{
			WorkspaceID: "workspace-1", ProjectID: "project-1", Limit: 100, Actor: lead,
		})
		if listErr != nil || len(page.Retrospectives) == 0 {
			t.Fatalf("list = %#v, error %v", page, listErr)
		}
		return counter.Load()
	}
	oneItemQueries := list()
	for index := 1; index < 20; index++ {
		create(index)
	}
	twentyItemQueries := list()
	if oneItemQueries != twentyItemQueries || twentyItemQueries > 7 {
		t.Fatalf("list query count = one item %d, twenty items %d, want same bound <= 7", oneItemQueries, twentyItemQueries)
	}
}

func TestProjectRetrospectiveRepositoryClaimsCompletesReplaysAndReauthorizesTarget(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	now := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	published := createPublishedProjectRetrospectiveForTarget(t, repository, "retro-target", lead, now)
	hash := strings.Repeat("a", 64)
	prepare := application.PrepareProjectRetrospectiveTargetCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: published.ID, ActionItemID: "action-target",
		TargetKind: "task", IdempotencyKey: "target-key-1", RequestHash: hash, Actor: lead, OccurredAt: now.Add(2 * time.Minute),
	}
	claim, err := repository.PrepareProjectRetrospectiveTarget(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	if claim.ActionItem.ID != "action-target" || claim.SourceRevision != 2 || claim.TargetKind != "task" || claim.TargetID != "" || claim.ChildIdempotencyKey == "" || len(claim.ChildIdempotencyKey) > 200 {
		t.Fatalf("claim = %#v", claim)
	}
	replayedClaim, err := repository.PrepareProjectRetrospectiveTarget(context.Background(), prepare)
	if err != nil || replayedClaim != claim {
		t.Fatalf("replayed claim = %#v, error %v", replayedClaim, err)
	}
	conflictingPrepare := prepare
	conflictingPrepare.RequestHash = strings.Repeat("b", 64)
	if _, err = repository.PrepareProjectRetrospectiveTarget(context.Background(), conflictingPrepare); !errors.Is(err, contract.ErrProjectRetrospectiveTargetConflict) {
		t.Fatalf("pending hash conflict = %v", err)
	}
	conflictingPrepare = prepare
	conflictingPrepare.TargetKind = "issue"
	if _, err = repository.PrepareProjectRetrospectiveTarget(context.Background(), conflictingPrepare); !errors.Is(err, contract.ErrProjectRetrospectiveTargetConflict) {
		t.Fatalf("pending kind conflict = %v", err)
	}
	if _, err = db.Exec(`UPDATE workspace_projects SET lead_id='owner-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.PrepareProjectRetrospectiveTarget(context.Background(), prepare); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("removed lead prepare retry = %v", err)
	}
	complete := application.CompleteProjectRetrospectiveTargetCommand{
		WorkspaceID: prepare.WorkspaceID, ProjectID: prepare.ProjectID, RetrospectiveID: prepare.RetrospectiveID, ActionItemID: prepare.ActionItemID,
		SourceRevision: claim.SourceRevision, TargetKind: claim.TargetKind, TargetID: "task-1",
		IdempotencyKey: prepare.IdempotencyKey, RequestHash: prepare.RequestHash, Actor: lead, OccurredAt: now.Add(3 * time.Minute),
	}
	if _, err = repository.CompleteProjectRetrospectiveTarget(context.Background(), complete); !errors.Is(err, contract.ErrWorkspacePermissionDenied) {
		t.Fatalf("removed lead complete = %v", err)
	}
	var pendingState string
	if err = db.QueryRow(`SELECT state FROM workspace_project_retrospective_action_links WHERE workspace_id='workspace-1' AND project_id='project-1' AND retrospective_id='retro-target' AND action_item_id='action-target'`).Scan(&pendingState); err != nil || pendingState != "pending" {
		t.Fatalf("pending state = %q, error %v", pendingState, err)
	}
	if _, err = db.Exec(`UPDATE workspace_projects SET lead_id='lead-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
		t.Fatal(err)
	}
	linked, err := repository.CompleteProjectRetrospectiveTarget(context.Background(), complete)
	if err != nil {
		t.Fatal(err)
	}
	if linked.ActionItemID != "action-target" || linked.SourceRevision != 2 || linked.State != "linked" || linked.TargetKind != "task" || linked.TargetID != "task-1" || linked.CreatedBy != "lead-1" {
		t.Fatalf("linked = %#v", linked)
	}
	replay, err := repository.CompleteProjectRetrospectiveTarget(context.Background(), complete)
	if err != nil || replay != linked {
		t.Fatalf("complete replay = %#v, error %v", replay, err)
	}
	conflictingComplete := complete
	conflictingComplete.RequestHash = strings.Repeat("c", 64)
	if _, err = repository.CompleteProjectRetrospectiveTarget(context.Background(), conflictingComplete); !errors.Is(err, contract.ErrIdempotencyConflict) {
		t.Fatalf("complete replay conflict = %v", err)
	}
	secondKeyPrepare := prepare
	secondKeyPrepare.IdempotencyKey = "target-key-2"
	secondClaim, err := repository.PrepareProjectRetrospectiveTarget(context.Background(), secondKeyPrepare)
	if err != nil || secondClaim.TargetID != "task-1" || secondClaim.ChildIdempotencyKey != claim.ChildIdempotencyKey {
		t.Fatalf("second key claim = %#v, error %v", secondClaim, err)
	}
	secondComplete := complete
	secondComplete.IdempotencyKey = secondKeyPrepare.IdempotencyKey
	secondReplay, err := repository.CompleteProjectRetrospectiveTarget(context.Background(), secondComplete)
	if err != nil || secondReplay != linked {
		t.Fatalf("second key complete = %#v, error %v", secondReplay, err)
	}
	var targetIdempotency, targetAudit, targetOutbox int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_mutation_idempotency WHERE workspace_id='workspace-1' AND action='workspace.project.retrospective.action_item.target'`).Scan(&targetIdempotency); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE workspace_id='workspace-1' AND action='workspace.project.retrospective.action_item_linked'`).Scan(&targetAudit); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_outbox_events WHERE workspace_id='workspace-1' AND event_type='retrospective:action_item_linked'`).Scan(&targetOutbox); err != nil {
		t.Fatal(err)
	}
	if targetIdempotency != 2 || targetAudit != 1 || targetOutbox != 1 {
		t.Fatalf("target effects = idempotency %d audit %d outbox %d", targetIdempotency, targetAudit, targetOutbox)
	}
}

func TestProjectRetrospectiveRepositoryTargetDoesNotOwnTaskOrIssuePersistence(t *testing.T) {
	source, err := os.ReadFile("project_retrospective_repository.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"workspace_todos", "workspace_issues"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("Project Retrospective repository directly references %s", forbidden)
		}
	}
}

func TestProjectRetrospectiveRepositoryConcurrentTargetCallersConvergeOnOneLink(t *testing.T) {
	db := openProjectRequirementDB(t)
	repository, err := persistence.NewProjectRetrospectiveRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	lead := contract.WorkspaceActor{Type: "member", ID: "lead-1"}
	now := time.Date(2026, 8, 20, 9, 30, 0, 0, time.UTC)
	createPublishedProjectRetrospectiveForTarget(t, repository, "retro-concurrent-target", lead, now)
	hash := strings.Repeat("7", 64)
	errorsByCaller := make([]error, 2)
	links := make([]contract.ProjectRetrospectiveActionLink, 2)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := range errorsByCaller {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			key := fmt.Sprintf("concurrent-target-%d", index)
			claim, claimErr := repository.PrepareProjectRetrospectiveTarget(context.Background(), application.PrepareProjectRetrospectiveTargetCommand{
				WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-concurrent-target", ActionItemID: "action-target",
				TargetKind: "task", IdempotencyKey: key, RequestHash: hash, Actor: lead, OccurredAt: now.Add(2 * time.Minute),
			})
			if claimErr != nil {
				errorsByCaller[index] = claimErr
				return
			}
			links[index], errorsByCaller[index] = repository.CompleteProjectRetrospectiveTarget(context.Background(), application.CompleteProjectRetrospectiveTargetCommand{
				WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: "retro-concurrent-target", ActionItemID: "action-target",
				SourceRevision: claim.SourceRevision, TargetKind: claim.TargetKind, TargetID: "task-shared",
				IdempotencyKey: key, RequestHash: hash, Actor: lead, OccurredAt: now.Add(3 * time.Minute),
			})
		}(index)
	}
	close(start)
	wait.Wait()
	for index, callErr := range errorsByCaller {
		if callErr != nil || links[index].TargetID != "task-shared" {
			t.Fatalf("caller %d link = %#v, error %v", index, links[index], callErr)
		}
	}
	var linksCount, targetIdempotency, targetAudit, targetOutbox int
	for query, target := range map[string]*int{
		`SELECT COUNT(*) FROM workspace_project_retrospective_action_links WHERE retrospective_id='retro-concurrent-target' AND state='linked'`:                           &linksCount,
		`SELECT COUNT(*) FROM workspace_mutation_idempotency WHERE action='workspace.project.retrospective.action_item.target' AND resource_id='retro-concurrent-target'`: &targetIdempotency,
		`SELECT COUNT(*) FROM workspace_audit_entries WHERE action='workspace.project.retrospective.action_item_linked' AND resource_id='retro-concurrent-target'`:        &targetAudit,
		`SELECT COUNT(*) FROM workspace_outbox_events WHERE event_type='retrospective:action_item_linked' AND aggregate_id='retro-concurrent-target'`:                     &targetOutbox,
	} {
		if err = db.QueryRow(query).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if linksCount != 1 || targetIdempotency != 2 || targetAudit != 1 || targetOutbox != 1 {
		t.Fatalf("concurrent effects = links %d idempotency %d audit %d outbox %d", linksCount, targetIdempotency, targetAudit, targetOutbox)
	}
}

func TestIssueRepositoryCreatesPrivateIdempotentTargetExactlyOnce(t *testing.T) {
	db := openProjectRequirementDB(t)
	base, err := persistence.NewIssueRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	repository, ok := base.(application.IdempotentIssueRepository)
	if !ok {
		t.Fatal("installed Issue repository does not implement private idempotent creation")
	}
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	newIssue := func(id string) issueDomain.Issue {
		t.Helper()
		value, newErr := issueDomain.New(id, "workspace-1", "Follow up", nil, "todo", "none", nil, nil, nil, nil, "member", "lead-1", 0, nil, nil, nil, nil, now)
		if newErr != nil {
			t.Fatal(newErr)
		}
		return value
	}
	hash := strings.Repeat("e", 64)
	created, replayed, err := repository.CreateIdempotently(context.Background(), application.IdempotentIssueCreateCommand{
		Value: newIssue("issue-target-1"), IdempotencyKey: "issue-target-key", RequestHash: hash,
	})
	if err != nil || replayed || created.ID != "issue-target-1" || created.Identifier != "ONE-1" {
		t.Fatalf("created Issue = %#v, replayed %t, error %v", created, replayed, err)
	}
	if _, err = db.Exec(`UPDATE workspace_issues SET title='Updated after create' WHERE workspace_id='workspace-1' AND id='issue-target-1'`); err != nil {
		t.Fatal(err)
	}
	replay, replayed, err := repository.CreateIdempotently(context.Background(), application.IdempotentIssueCreateCommand{
		Value: newIssue("issue-target-2"), IdempotencyKey: "issue-target-key", RequestHash: hash,
	})
	if err != nil || !replayed || replay.ID != "issue-target-1" || replay.Title != "Updated after create" {
		t.Fatalf("replayed Issue = %#v, replayed %t, error %v", replay, replayed, err)
	}
	if _, _, err = repository.CreateIdempotently(context.Background(), application.IdempotentIssueCreateCommand{
		Value: newIssue("issue-target-3"), IdempotencyKey: "issue-target-key", RequestHash: strings.Repeat("f", 64),
	}); !errors.Is(err, contract.ErrIdempotencyConflict) {
		t.Fatalf("Issue idempotency conflict = %v", err)
	}
	var issues, idempotency, nextNumber int
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_issues WHERE workspace_id='workspace-1'`).Scan(&issues); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_mutation_idempotency WHERE workspace_id='workspace-1' AND action='workspace.issue.create.idempotent'`).Scan(&idempotency); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT next_issue_number FROM workspaces WHERE id='workspace-1'`).Scan(&nextNumber); err != nil {
		t.Fatal(err)
	}
	if issues != 1 || idempotency != 1 || nextNumber != 2 {
		t.Fatalf("Issue effects = rows %d idempotency %d next number %d", issues, idempotency, nextNumber)
	}
}

func createPublishedProjectRetrospectiveForTarget(t *testing.T, repository *persistence.ProjectRetrospectiveRepository, retrospectiveID string, actor contract.WorkspaceActor, now time.Time) contract.ProjectRetrospective {
	t.Helper()
	created, err := repository.CreateProjectRetrospective(context.Background(), application.CreateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: retrospectiveID,
		Content: retrospectiveDomain.Content{
			Summary: "Published target", Lessons: []string{"Keep source provenance"},
			ActionItems: []retrospectiveDomain.ActionItem{{ID: "action-target", Title: "Follow up", Description: "Close the loop", AssigneeID: "ordinary-member", DueDate: "2026-08-30"}},
		},
		Participants:   []retrospectiveDomain.Participant{{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant}},
		IdempotencyKey: "create-" + retrospectiveID, RequestHash: strings.Repeat("d", 64), Actor: actor, OccurredAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := repository.MutateProjectRetrospective(context.Background(), application.MutateProjectRetrospectiveCommand{
		WorkspaceID: "workspace-1", ProjectID: "project-1", RetrospectiveID: created.ID,
		ExpectedRevision: 1, Action: retrospectiveDomain.ActionPublish, RequestID: "publish-" + retrospectiveID, Actor: actor, OccurredAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return published
}
