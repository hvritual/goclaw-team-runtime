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

	workspace "github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func TestKnowledgeReviewRepositoryIdempotencyLifecycleAndAtomicPublication(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-review.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewKnowledgeReviewRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	candidate := contract.KnowledgeCandidate{ID: "candidate-1", WorkspaceID: "workspace-1", Kind: "lesson", Title: "Retain evidence", Content: "Keep exact evidence.", Reason: "Accepted behavior", Status: "candidate", Revision: 1, ProposedBy: "user-1", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano), SourceRefs: []contract.KnowledgeSourceRef{{Type: "acceptance_conclusion", ID: "issue-1", Revision: "sha256:abc", Citation: "Acceptance passed"}}}
	created, err := repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "proposal-key", RequestHash: strings.Repeat("a", 64), AuditID: "audit-create"})
	if err != nil || created.Replayed {
		t.Fatalf("create = %#v, %v", created, err)
	}
	replay, err := repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: contract.KnowledgeCandidate{ID: "different", WorkspaceID: "workspace-1"}, IdempotencyKey: "proposal-key", RequestHash: strings.Repeat("a", 64), AuditID: "other"})
	if err != nil || !replay.Replayed || replay.Candidate.ID != "candidate-1" {
		t.Fatalf("replay = %#v, %v", replay, err)
	}
	if _, err = repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "proposal-key", RequestHash: strings.Repeat("b", 64), AuditID: "other"}); !errors.Is(err, contract.ErrKnowledgeIdempotencyConflict) {
		t.Fatalf("conflicting replay = %v", err)
	}
	if _, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-1", Action: "approve", Rationale: "self", ActorID: "user-1", AuditID: "audit-self", PublicationID: "unused", ExpectedRevision: 1, OccurredAt: now.Add(time.Minute)}); !errors.Is(err, contract.ErrKnowledgeSelfReview) {
		t.Fatalf("self review = %v", err)
	}
	approved, err := repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-1", Action: "approve", Rationale: "independent approval", ActorID: "user-2", AuditID: "audit-approve", PublicationID: "unused", ExpectedRevision: 1, OccurredAt: now.Add(time.Minute)})
	if err != nil || approved.Candidate.Status != "in_review" || approved.Candidate.Revision != 2 {
		t.Fatalf("approve = %#v, %v", approved, err)
	}
	if _, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-1", Action: "publish", Rationale: "stale", ActorID: "user-2", AuditID: "audit-stale", PublicationID: "knowledge-1", ExpectedRevision: 1, OccurredAt: now.Add(2 * time.Minute)}); !errors.Is(err, contract.ErrKnowledgeRevisionConflict) {
		t.Fatalf("stale publish = %v", err)
	}
	published, err := repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-1", Action: "publish", Rationale: "source verified", ActorID: "user-2", AuditID: "audit-publish", PublicationID: "knowledge-1", ExpectedRevision: 2, OccurredAt: now.Add(2 * time.Minute)})
	if err != nil || published.Entry == nil || published.Entry.ID != "knowledge-1" || published.Candidate.Status != "published" {
		t.Fatalf("publish = %#v, %v", published, err)
	}
	var candidates, reviews, audits, entries, sources int
	for query, target := range map[string]*int{`SELECT COUNT(*) FROM workspace_knowledge_candidates`: &candidates, `SELECT COUNT(*) FROM workspace_knowledge_review_events`: &reviews, `SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_kind='knowledge_candidate'`: &audits, `SELECT COUNT(*) FROM workspace_governed_knowledge WHERE id='knowledge-1' AND status='published'`: &entries, `SELECT COUNT(*) FROM workspace_knowledge_source_refs WHERE knowledge_id='knowledge-1'`: &sources} {
		if err = db.QueryRow(query).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if candidates != 1 || reviews != 2 || audits != 3 || entries != 1 || sources != 1 {
		t.Fatalf("counts candidate=%d review=%d audit=%d entry=%d source=%d", candidates, reviews, audits, entries, sources)
	}
}

func TestKnowledgeReviewRepositorySerializesExpectedRevision(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-review-concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = db.Close() })
	if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository, _ := persistence.NewKnowledgeReviewRepository(persistence.Config{DB: db})
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	candidate := contract.KnowledgeCandidate{ID: "candidate-concurrent", WorkspaceID: "workspace-1", Kind: "lesson", Title: "Concurrent", Content: "Body", Reason: "Reason", Status: "candidate", Revision: 1, ProposedBy: "user-1", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano)}
	if _, err = repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "concurrent-key", RequestHash: strings.Repeat("d", 64), AuditID: "audit-create"}); err != nil {
		t.Fatal(err)
	}
	errorsSeen := make([]error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	for i := 0; i < 2; i++ {
		go func(index int) {
			defer wait.Done()
			errorsSeen[index] = func() error {
				_, reviewErr := repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-concurrent", Action: "approve", Rationale: "independent", ActorID: []string{"user-2", "user-3"}[index], AuditID: []string{"audit-2", "audit-3"}[index], PublicationID: "unused", ExpectedRevision: 1, OccurredAt: now.Add(time.Minute)})
				return reviewErr
			}()
		}(i)
	}
	wait.Wait()
	successes, conflicts := 0, 0
	for _, seen := range errorsSeen {
		if seen == nil {
			successes++
		} else if errors.Is(seen, contract.ErrKnowledgeRevisionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent error = %v", seen)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success/conflict = %d/%d (%v)", successes, conflicts, errorsSeen)
	}
	var revision, audits int
	if err = db.QueryRow(`SELECT revision FROM workspace_knowledge_candidates WHERE id='candidate-concurrent'`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_id='candidate-concurrent'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if revision != 2 || audits != 2 {
		t.Fatalf("revision/audits = %d/%d", revision, audits)
	}
}

func TestKnowledgeReviewRepositoryCancelledProposalLeavesNoRows(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-review-cancelled.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository, err := persistence.NewKnowledgeReviewRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	now := time.Date(2026, 8, 18, 14, 30, 0, 0, time.UTC).Format(time.RFC3339Nano)
	candidate := contract.KnowledgeCandidate{ID: "candidate-cancelled", WorkspaceID: "workspace-1", Kind: "lesson", Title: "Cancelled", Content: "Body", Reason: "Reason", Status: "candidate", Revision: 1, ProposedBy: "user-1", CreatedAt: now, UpdatedAt: now}
	if _, err = repository.CreateKnowledgeCandidate(ctx, application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "cancelled-key", RequestHash: strings.Repeat("a", 64), AuditID: "audit-cancelled"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled proposal = %v", err)
	}
	for _, table := range []string{"workspace_knowledge_candidates", "workspace_audit_entries", "workspace_mutation_idempotency"} {
		var count int
		if err = db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s rows = %d, %v", table, count, err)
		}
	}
}

func TestKnowledgeReviewRepositoryNonPublicationTransitions(t *testing.T) {
	tests := []struct {
		name       string
		actions    []string
		wantStatus string
		wantRev    int
	}{
		{name: "reject", actions: []string{"approve", "reject"}, wantStatus: "rejected", wantRev: 3},
		{name: "quarantine and return", actions: []string{"approve", "quarantine", "return"}, wantStatus: "in_review", wantRev: 4},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-review-transitions.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			repository, err := persistence.NewKnowledgeReviewRepository(persistence.Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 8, 18, 14, 45, 0, 0, time.UTC)
			candidate := contract.KnowledgeCandidate{ID: "candidate-transitions", WorkspaceID: "workspace-1", Kind: "lesson", Title: "Transitions", Content: "Body", Reason: "Reason", Status: "candidate", Revision: 1, ProposedBy: "user-1", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano)}
			if _, err = repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "transitions-key", RequestHash: strings.Repeat("b", 64), AuditID: "audit-create"}); err != nil {
				t.Fatal(err)
			}
			for index, action := range test.actions {
				result, reviewErr := repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: candidate.ID, Action: action, Rationale: "independent review", ActorID: "user-2", AuditID: "audit-" + action, PublicationID: "unused", ExpectedRevision: index + 1, OccurredAt: now.Add(time.Duration(index+1) * time.Minute)})
				if reviewErr != nil {
					t.Fatalf("%s = %v", action, reviewErr)
				}
				if result.Candidate.Revision != index+2 {
					t.Fatalf("%s revision = %d", action, result.Candidate.Revision)
				}
			}
			var status string
			var revision, reviews, audits int
			if err = db.QueryRow(`SELECT status,revision FROM workspace_knowledge_candidates WHERE id=?`, candidate.ID).Scan(&status, &revision); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_knowledge_review_events WHERE candidate_id=?`, candidate.ID).Scan(&reviews); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE resource_id=?`, candidate.ID).Scan(&audits); err != nil {
				t.Fatal(err)
			}
			if status != test.wantStatus || revision != test.wantRev || reviews != len(test.actions) || audits != len(test.actions)+1 {
				t.Fatalf("status/revision/reviews/audits = %s/%d/%d/%d", status, revision, reviews, audits)
			}
		})
	}
}

func TestKnowledgeReviewRepositoryAuditFailureRollsBackPublication(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-review-audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository, _ := persistence.NewKnowledgeReviewRepository(persistence.Config{DB: db})
	now := time.Date(2026, 8, 18, 15, 0, 0, 0, time.UTC)
	candidate := contract.KnowledgeCandidate{ID: "candidate-audit", WorkspaceID: "workspace-1", Kind: "lesson", Title: "Audit", Content: "Body", Reason: "Reason", Status: "candidate", Revision: 1, ProposedBy: "user-1", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano), SourceRefs: []contract.KnowledgeSourceRef{{Type: "acceptance_conclusion", ID: "issue-1", Revision: "sha256:audit", Citation: "Accepted"}}}
	if _, err = repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "audit-key", RequestHash: strings.Repeat("e", 64), AuditID: "audit-create"}); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-audit", Action: "approve", Rationale: "approved", ActorID: "user-2", AuditID: "audit-approve", PublicationID: "unused", ExpectedRevision: 1, OccurredAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_knowledge_publish_audit BEFORE INSERT ON workspace_audit_entries WHEN NEW.action='workspace.knowledge.publish' BEGIN SELECT RAISE(ABORT,'audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "candidate-audit", Action: "publish", Rationale: "publish", ActorID: "user-2", AuditID: "audit-publish", PublicationID: "knowledge-audit", ExpectedRevision: 2, OccurredAt: now.Add(2 * time.Minute)}); err == nil {
		t.Fatal("publish succeeded despite audit failure")
	}
	var status string
	var revision, entries, publications, reviews int
	if err = db.QueryRow(`SELECT status,revision FROM workspace_knowledge_candidates WHERE id='candidate-audit'`).Scan(&status, &revision); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_governed_knowledge WHERE id='knowledge-audit'`).Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_knowledge_publications WHERE candidate_id='candidate-audit'`).Scan(&publications); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM workspace_knowledge_review_events WHERE candidate_id='candidate-audit'`).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if status != "in_review" || revision != 2 || entries != 0 || publications != 0 || reviews != 1 {
		t.Fatalf("rollback status=%s revision=%d entries=%d publications=%d reviews=%d", status, revision, entries, publications, reviews)
	}
}

func TestKnowledgeReviewRepositorySupersedesAndInvalidatesExactTargets(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "knowledge-targets.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('target-1','workspace-1','lesson','published',1,'2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`, `INSERT INTO workspace_knowledge_revisions(knowledge_id,revision,supersedes_revision,title,content,created_by,created_at) VALUES('target-1',1,0,'Old','Old body','user-0','2026-08-18T00:00:00Z')`} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repository, _ := persistence.NewKnowledgeReviewRepository(persistence.Config{DB: db})
	target := "target-1"
	now := time.Date(2026, 8, 18, 13, 0, 0, 0, time.UTC)
	candidate := contract.KnowledgeCandidate{ID: "replace-1", WorkspaceID: "workspace-1", KnowledgeID: &target, Kind: "lesson", Title: "New", Content: "New body", Reason: "Replace", Status: "candidate", Revision: 1, ProposedBy: "user-1", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano), SourceRefs: []contract.KnowledgeSourceRef{{Type: "acceptance_conclusion", ID: "issue-1", Revision: "sha256:new", Citation: "New acceptance"}}}
	created, err := repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: "replace-key", RequestHash: strings.Repeat("c", 64), AuditID: "audit-create"})
	if err != nil || created.Candidate.TargetRevision != 1 {
		t.Fatalf("target capture = %#v, %v", created, err)
	}
	_, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "replace-1", Action: "approve", Rationale: "approve", ActorID: "user-2", AuditID: "audit-approve", PublicationID: "unused", ExpectedRevision: 1, OccurredAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	replaced, err := repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "replace-1", Action: "supersede", Rationale: "replace exact target", ActorID: "user-2", AuditID: "audit-replace", PublicationID: "replacement-1", ExpectedRevision: 2, OccurredAt: now.Add(2 * time.Minute)})
	if err != nil || replaced.Entry == nil || replaced.Entry.ID != "replacement-1" {
		t.Fatalf("supersede = %#v, %v", replaced, err)
	}
	var oldStatus, newStatus string
	if err = db.QueryRow(`SELECT status FROM workspace_governed_knowledge WHERE id='target-1'`).Scan(&oldStatus); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT status FROM workspace_governed_knowledge WHERE id='replacement-1'`).Scan(&newStatus); err != nil {
		t.Fatal(err)
	}
	if oldStatus != "superseded" || newStatus != "published" {
		t.Fatalf("statuses = %s/%s", oldStatus, newStatus)
	}
	if _, err = db.Exec(`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('target-2','workspace-1','lesson','published',1,'2026-08-18T00:00:00Z','2026-08-18T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	target2 := "target-2"
	invalidateCandidate := candidate
	invalidateCandidate.ID, invalidateCandidate.KnowledgeID, invalidateCandidate.Title = "invalidate-1", &target2, "Invalidate"
	created, err = repository.CreateKnowledgeCandidate(context.Background(), application.CreateKnowledgeCandidateCommand{Candidate: invalidateCandidate, IdempotencyKey: "invalidate-key", RequestHash: strings.Repeat("f", 64), AuditID: "audit-invalidate-create"})
	if err != nil || created.Candidate.TargetRevision != 1 {
		t.Fatalf("invalidate capture = %#v, %v", created, err)
	}
	if _, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "invalidate-1", Action: "approve", Rationale: "approve invalidation", ActorID: "user-2", AuditID: "audit-invalidate-approve", PublicationID: "unused", ExpectedRevision: 1, OccurredAt: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE workspace_governed_knowledge SET current_revision=2 WHERE id='target-2'`); err != nil {
		t.Fatal(err)
	}
	_, err = repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "invalidate-1", Action: "invalidate", Rationale: "stale target", ActorID: "user-2", AuditID: "audit-invalidate", PublicationID: "unused", ExpectedRevision: 2, OccurredAt: now.Add(4 * time.Minute)})
	var conflict *contract.KnowledgeRevisionConflictError
	if !errors.As(err, &conflict) || conflict.Resource != "knowledge" || conflict.CurrentRevision != 2 {
		t.Fatalf("target conflict = %#v, %v", conflict, err)
	}
	if _, err = db.Exec(`UPDATE workspace_governed_knowledge SET current_revision=1 WHERE id='target-2'`); err != nil {
		t.Fatal(err)
	}
	invalidated, err := repository.ReviewKnowledgeCandidate(context.Background(), application.ReviewKnowledgeCandidateCommand{WorkspaceID: "workspace-1", CandidateID: "invalidate-1", Action: "invalidate", Rationale: "invalidate exact target", ActorID: "user-2", AuditID: "audit-invalidate-final", PublicationID: "unused", ExpectedRevision: 2, OccurredAt: now.Add(5 * time.Minute)})
	if err != nil || invalidated.Entry != nil {
		t.Fatalf("invalidate = %#v, %v", invalidated, err)
	}
	if err = db.QueryRow(`SELECT status FROM workspace_governed_knowledge WHERE id='target-2'`).Scan(&oldStatus); err != nil || oldStatus != "invalidated" {
		t.Fatalf("invalidated target = %s, %v", oldStatus, err)
	}
}
