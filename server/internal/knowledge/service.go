package knowledge

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrReasonRequired    = errors.New("proposal reason is required")
	ErrRationaleRequired = errors.New("review rationale is required")
	ErrReviewerRequired  = errors.New("reviewer is required")
	ErrStoreRequired     = errors.New("knowledge store is required")
	ErrNotFound          = errors.New("knowledge record not found")
	ErrWorkspaceMismatch = errors.New("knowledge workspace mismatch")
	ErrRevisionConflict  = errors.New("knowledge revision conflict")
	ErrInvalidReview     = errors.New("invalid knowledge review action")
	ErrInvalidEvidence   = errors.New("invalid knowledge evidence")
	ErrInvalidProposal   = errors.New("invalid knowledge proposal")
)

type Repository interface {
	CreateCandidate(context.Context, Candidate) (Candidate, error)
	GetCandidate(context.Context, string) (Candidate, error)
	ListCandidates(context.Context, CandidateQuery) (CandidatePage, error)
	GetEntry(context.Context, string, string) (Entry, error)
	ReviewCandidate(context.Context, ReviewCommand) (Candidate, *Entry, error)
	IngestEvidence(context.Context, IngestionCommand) (IngestionResult, error)
}

type PromotionPolicy interface {
	Decide(Evidence) PromotionDecision
}

type SearchIndex interface {
	Search(context.Context, SearchQuery) (SearchPage, error)
	Rebuild(context.Context) error
}

type DefaultPromotionPolicy struct{}

func (DefaultPromotionPolicy) Decide(evidence Evidence) PromotionDecision {
	if evidence.HasConflict || evidence.Confidence < 0.8 || len(evidence.SourceRefs) == 0 {
		return PromotionDecision{
			Action: PromotionQuarantine,
			Reason: "Evidence is conflicting, low-confidence, or missing provenance.",
		}
	}
	switch evidence.Kind {
	case KindGoal, KindDecision, KindConstraint, KindRequirement, KindProcedure, KindLesson:
		return PromotionDecision{
			Action: PromotionCandidate,
			Reason: "This knowledge kind requires human review.",
		}
	default:
		if evidence.Terminal && evidence.Validated {
			return PromotionDecision{
				Action: PromotionPublish,
				Reason: "A deterministic terminal fact passed validation.",
			}
		}
		return PromotionDecision{
			Action: PromotionCandidate,
			Reason: "Evidence is not terminal and validated.",
		}
	}
}

type Service struct {
	store  Repository
	policy PromotionPolicy
}

func NewService(store Repository, policy PromotionPolicy) *Service {
	if policy == nil {
		policy = DefaultPromotionPolicy{}
	}
	return &Service{store: store, policy: policy}
}

func (s *Service) Propose(ctx context.Context, input ProposalInput) (Candidate, error) {
	if strings.TrimSpace(input.Reason) == "" {
		return Candidate{}, ErrReasonRequired
	}
	if strings.TrimSpace(input.WorkspaceID) == "" ||
		strings.TrimSpace(input.Title) == "" ||
		strings.TrimSpace(input.Content) == "" ||
		strings.TrimSpace(input.ProposedBy) == "" ||
		!validKind(input.Kind) {
		return Candidate{}, ErrInvalidProposal
	}
	if s.store == nil {
		return Candidate{}, ErrStoreRequired
	}
	return s.store.CreateCandidate(ctx, Candidate{
		WorkspaceID: strings.TrimSpace(input.WorkspaceID),
		ProjectID:   strings.TrimSpace(input.ProjectID),
		Kind:        input.Kind,
		Title:       strings.TrimSpace(input.Title),
		Content:     strings.TrimSpace(input.Content),
		Reason:      strings.TrimSpace(input.Reason),
		Status:      StatusCandidate,
		Revision:    1,
		ProposedBy:  strings.TrimSpace(input.ProposedBy),
		SourceRefs:  input.SourceRefs,
	})
}

func validKind(kind Kind) bool {
	switch kind {
	case KindGoal, KindDecision, KindConstraint, KindRequirement, KindProcedure, KindLesson, KindReference:
		return true
	default:
		return false
	}
}

func (s *Service) IngestEvidence(ctx context.Context, evidence Evidence) (IngestionResult, error) {
	if s.store == nil {
		return IngestionResult{}, ErrStoreRequired
	}
	if strings.TrimSpace(evidence.ID) == "" ||
		strings.TrimSpace(evidence.WorkspaceID) == "" ||
		strings.TrimSpace(evidence.SourceType) == "" ||
		strings.TrimSpace(evidence.SourceID) == "" ||
		strings.TrimSpace(evidence.IdempotencyKey) == "" ||
		strings.TrimSpace(evidence.Content) == "" {
		return IngestionResult{}, ErrInvalidEvidence
	}
	evidence.ID = strings.TrimSpace(evidence.ID)
	evidence.WorkspaceID = strings.TrimSpace(evidence.WorkspaceID)
	evidence.ProjectID = strings.TrimSpace(evidence.ProjectID)
	evidence.SourceType = strings.TrimSpace(evidence.SourceType)
	evidence.SourceID = strings.TrimSpace(evidence.SourceID)
	evidence.IdempotencyKey = strings.TrimSpace(evidence.IdempotencyKey)
	evidence.SourceRefs = append([]SourceRef(nil), evidence.SourceRefs...)

	currentTime := time.Now().UTC()
	decision := s.policy.Decide(evidence)
	command := IngestionCommand{Evidence: evidence}
	switch decision.Action {
	case PromotionCandidate, PromotionQuarantine:
		status := StatusCandidate
		if decision.Action == PromotionQuarantine {
			status = StatusQuarantined
		}
		command.Candidate = &Candidate{
			ID:          uuid.NewString(),
			WorkspaceID: evidence.WorkspaceID,
			ProjectID:   evidence.ProjectID,
			Kind:        evidence.Kind,
			Title:       strings.TrimSpace(evidence.Title),
			Content:     strings.TrimSpace(evidence.Content),
			Reason:      decision.Reason,
			Status:      status,
			Revision:    1,
			ProposedBy:  strings.TrimSpace(evidence.ActorID),
			SourceRefs:  append([]SourceRef(nil), evidence.SourceRefs...),
			CreatedAt:   currentTime,
			UpdatedAt:   currentTime,
		}
	case PromotionPublish:
		command.Entry = &Entry{
			ID:              uuid.NewString(),
			WorkspaceID:     evidence.WorkspaceID,
			ProjectID:       evidence.ProjectID,
			Kind:            evidence.Kind,
			Status:          StatusPublished,
			CurrentRevision: 1,
			Revisions: []Revision{{
				Number:     1,
				Title:      strings.TrimSpace(evidence.Title),
				Content:    strings.TrimSpace(evidence.Content),
				CreatedBy:  strings.TrimSpace(evidence.ActorID),
				CreatedAt:  currentTime,
				SourceRefs: append([]SourceRef(nil), evidence.SourceRefs...),
			}},
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
		}
	default:
		return IngestionResult{}, ErrInvalidEvidence
	}
	return s.store.IngestEvidence(ctx, command)
}

func (s *Service) Review(ctx context.Context, input ReviewInput) (Candidate, *Entry, error) {
	if s.store == nil {
		return Candidate{}, nil, ErrStoreRequired
	}
	if strings.TrimSpace(input.ReviewerID) == "" {
		return Candidate{}, nil, ErrReviewerRequired
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return Candidate{}, nil, ErrRationaleRequired
	}
	candidate, err := s.store.GetCandidate(ctx, input.CandidateID)
	if err != nil {
		return Candidate{}, nil, err
	}
	if candidate.WorkspaceID != strings.TrimSpace(input.WorkspaceID) {
		return Candidate{}, nil, ErrWorkspaceMismatch
	}

	var newStatus Status
	switch input.Action {
	case ReviewApprove:
		newStatus = StatusPublished
	case ReviewReject:
		newStatus = StatusRejected
	case ReviewQuarantine:
		newStatus = StatusQuarantined
	default:
		return Candidate{}, nil, ErrInvalidReview
	}

	currentTime := time.Now().UTC()
	review := Review{
		Action:      input.Action,
		ReviewerID:  strings.TrimSpace(input.ReviewerID),
		Rationale:   strings.TrimSpace(input.Rationale),
		ReviewedAt:  currentTime,
		OldRevision: candidate.Revision,
		NewRevision: candidate.Revision + 1,
	}
	var entry *Entry
	if input.Action == ReviewApprove {
		entry = &Entry{
			ID:              uuid.NewString(),
			WorkspaceID:     candidate.WorkspaceID,
			ProjectID:       candidate.ProjectID,
			CandidateID:     candidate.ID,
			Kind:            candidate.Kind,
			Status:          StatusPublished,
			CurrentRevision: 1,
			Revisions: []Revision{{
				Number:     1,
				Title:      candidate.Title,
				Content:    candidate.Content,
				CreatedBy:  review.ReviewerID,
				CreatedAt:  currentTime,
				SourceRefs: append([]SourceRef(nil), candidate.SourceRefs...),
			}},
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
		}
	}
	return s.store.ReviewCandidate(ctx, ReviewCommand{
		CandidateID:      candidate.ID,
		WorkspaceID:      candidate.WorkspaceID,
		ExpectedRevision: input.ExpectedRevision,
		NewStatus:        newStatus,
		Review:           review,
		Entry:            entry,
	})
}
