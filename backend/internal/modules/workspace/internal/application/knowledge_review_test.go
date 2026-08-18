package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type knowledgeReviewRepositoryStub struct {
	replay            bool
	created, reviewed int
	candidate         contract.KnowledgeCandidate
	listed            []contract.KnowledgeCandidate
}

func (s *knowledgeReviewRepositoryStub) CreateKnowledgeCandidate(_ context.Context, command CreateKnowledgeCandidateCommand) (CreatedKnowledgeCandidate, error) {
	s.created++
	s.candidate = command.Candidate
	return CreatedKnowledgeCandidate{Candidate: command.Candidate, Replayed: s.replay}, nil
}
func (s *knowledgeReviewRepositoryStub) ListKnowledgeCandidates(context.Context, string) ([]contract.KnowledgeCandidate, error) {
	return s.listed, nil
}

func TestKnowledgeReviewUseCaseEmptyQueueIsAnArray(t *testing.T) {
	repository := &knowledgeReviewRepositoryStub{}
	service, err := NewKnowledgeReviewUseCase(repository, knowledgeReviewAuthorizerStub{denied: map[string]error{}}, knowledgeReviewAssetStub{belongs: true}, func(context.Context) (string, error) { return "id", nil }, time.Now, []byte("01234567890123456789012345678901"), nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ListKnowledgeCandidates(context.Background(), contract.ListKnowledgeCandidatesRequest{WorkspaceID: "workspace-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates == nil || len(result.Candidates) != 0 {
		t.Fatalf("empty candidates = %#v", result.Candidates)
	}
}
func (s *knowledgeReviewRepositoryStub) ReviewKnowledgeCandidate(_ context.Context, command ReviewKnowledgeCandidateCommand) (contract.ReviewKnowledgeResponse, error) {
	s.reviewed++
	return contract.ReviewKnowledgeResponse{Candidate: contract.KnowledgeCandidate{ID: command.CandidateID, WorkspaceID: command.WorkspaceID, Status: "in_review", Revision: command.ExpectedRevision + 1}}, nil
}

type knowledgeReviewAuthorizerStub struct{ denied map[string]error }

func (s knowledgeReviewAuthorizerStub) AuthorizeWorkspace(_ context.Context, _ string, permission string) error {
	return s.denied[permission]
}

type knowledgeReviewAssetStub struct{ belongs bool }

func (s knowledgeReviewAssetStub) AssetBelongsToWorkspace(context.Context, string, string) (bool, error) {
	return s.belongs, nil
}

type knowledgeReviewEventsStub struct{ count int }

func (s *knowledgeReviewEventsStub) Publish(string, string, any, string, string) { s.count++ }

func TestKnowledgeReviewUseCaseProposalReplayDoesNotDuplicateRealtime(t *testing.T) {
	repository := &knowledgeReviewRepositoryStub{}
	events := &knowledgeReviewEventsStub{}
	ids := 0
	service, err := NewKnowledgeReviewUseCase(repository, knowledgeReviewAuthorizerStub{denied: map[string]error{}}, knowledgeReviewAssetStub{belongs: true}, func(context.Context) (string, error) { ids++; return "id-" + string(rune('0'+ids)), nil }, func() time.Time { return time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC) }, []byte("01234567890123456789012345678901"), events)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
	request := contract.ProposeKnowledgeRequest{WorkspaceID: "workspace-1", IdempotencyKey: "key-1", Kind: "lesson", Title: "Title", Content: "Body", Reason: "Reason", SourceRefs: []contract.KnowledgeSourceRef{{Type: "acceptance_conclusion", ID: "issue-1", Revision: "sha256:abc", Citation: "Accepted"}}}
	if _, err = service.ProposeKnowledge(ctx, request); err != nil {
		t.Fatal(err)
	}
	repository.replay = true
	if _, err = service.ProposeKnowledge(ctx, request); err != nil {
		t.Fatal(err)
	}
	if events.count != 1 {
		t.Fatalf("realtime count = %d", events.count)
	}
}

func TestKnowledgeReviewUseCaseOwnerOverrideIsExplicitAndReasoned(t *testing.T) {
	repository := &knowledgeReviewRepositoryStub{}
	events := &knowledgeReviewEventsStub{}
	service, err := NewKnowledgeReviewUseCase(repository, knowledgeReviewAuthorizerStub{denied: map[string]error{}}, knowledgeReviewAssetStub{belongs: true}, func(context.Context) (string, error) { return "id", nil }, time.Now, []byte("01234567890123456789012345678901"), events)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "owner-1")
	short := contract.ReviewKnowledgeRequest{WorkspaceID: "workspace-1", CandidateID: "candidate-1", Action: "approve", ExpectedRevision: 1, Rationale: "too short", Emergency: true}
	if _, err = service.ReviewKnowledge(ctx, short); !errors.Is(err, contract.ErrInvalidKnowledgeReview) {
		t.Fatalf("short emergency = %v", err)
	}
	service.authorizer = knowledgeReviewAuthorizerStub{denied: map[string]error{contract.PermissionKnowledgeSelfReviewOverride: contract.ErrWorkspacePermissionDenied}}
	short.Rationale = "documented emergency reason"
	if _, err = service.ReviewKnowledge(ctx, short); !errors.Is(err, contract.ErrKnowledgeSelfReview) {
		t.Fatalf("admin override = %v", err)
	}
	service.authorizer = knowledgeReviewAuthorizerStub{denied: map[string]error{}}
	if _, err = service.ReviewKnowledge(ctx, short); err != nil {
		t.Fatal(err)
	}
	if repository.reviewed != 1 || events.count != 1 {
		t.Fatalf("review/event = %d/%d", repository.reviewed, events.count)
	}
}

func TestKnowledgeReviewUseCaseRejectsForeignAssetBeforeRepository(t *testing.T) {
	repository := &knowledgeReviewRepositoryStub{}
	service, _ := NewKnowledgeReviewUseCase(repository, knowledgeReviewAuthorizerStub{denied: map[string]error{}}, knowledgeReviewAssetStub{belongs: false}, func(context.Context) (string, error) { return "id", nil }, time.Now, []byte("01234567890123456789012345678901"), nil)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
	asset, version := "asset-1", "version-1"
	_, err := service.ProposeKnowledge(ctx, contract.ProposeKnowledgeRequest{WorkspaceID: "workspace-1", IdempotencyKey: "key", Kind: "lesson", Title: "Title", Content: "Body", Reason: "Reason", SourceRefs: []contract.KnowledgeSourceRef{{Type: "attachment", ID: "asset-1", Revision: "version-1", Citation: "Evidence", AssetID: &asset, AssetVersionID: &version}}})
	if !errors.Is(err, contract.ErrAssetOutsideWorkspace) || repository.created != 0 {
		t.Fatalf("foreign asset = %v, creates=%d", err, repository.created)
	}
}
