package contract

import (
	"context"
	"errors"
)

var (
	ErrInvalidKnowledgeReview       = errors.New("invalid Knowledge review")
	ErrKnowledgeCandidateNotFound   = errors.New("Knowledge candidate not found")
	ErrKnowledgeRevisionConflict    = errors.New("Knowledge revision conflict")
	ErrKnowledgeIdempotencyConflict = errors.New("Knowledge idempotency conflict")
	ErrKnowledgeSelfReview          = errors.New("Knowledge self-review denied")
)

type KnowledgeRevisionConflictError struct {
	Resource        string
	CurrentRevision int
}

func (e *KnowledgeRevisionConflictError) Error() string { return ErrKnowledgeRevisionConflict.Error() }
func (e *KnowledgeRevisionConflictError) Unwrap() error { return ErrKnowledgeRevisionConflict }

type KnowledgeCandidate struct {
	ID             string               `json:"id"`
	WorkspaceID    string               `json:"workspace_id"`
	ProjectID      *string              `json:"project_id"`
	KnowledgeID    *string              `json:"knowledge_id"`
	TargetRevision int                  `json:"target_revision"`
	Kind           string               `json:"kind"`
	Title          string               `json:"title"`
	Content        string               `json:"content"`
	Reason         string               `json:"reason"`
	Status         string               `json:"status"`
	Revision       int                  `json:"revision"`
	ProposedBy     string               `json:"proposed_by"`
	SourceRefs     []KnowledgeSourceRef `json:"source_refs"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
}

type ProposeKnowledgeRequest struct {
	WorkspaceID    string
	IdempotencyKey string
	ProjectID      *string              `json:"project_id"`
	KnowledgeID    *string              `json:"knowledge_id"`
	Kind           string               `json:"kind"`
	Title          string               `json:"title"`
	Content        string               `json:"content"`
	Reason         string               `json:"reason"`
	SourceRefs     []KnowledgeSourceRef `json:"source_refs"`
}

type ListKnowledgeCandidatesRequest struct {
	WorkspaceID, Cursor string
	Limit               int
}
type KnowledgeCandidateListResponse struct {
	Candidates []KnowledgeCandidate `json:"candidates"`
	Total      int                  `json:"total"`
	NextCursor *string              `json:"next_cursor"`
}
type ReviewKnowledgeRequest struct {
	WorkspaceID, CandidateID, Action, Rationale string
	ExpectedRevision                            int
	Emergency                                   bool
}
type ReviewKnowledgeResponse struct {
	Candidate KnowledgeCandidate      `json:"candidate"`
	Entry     *GovernedKnowledgeEntry `json:"entry"`
}

type KnowledgeReviewService interface {
	ProposeKnowledge(context.Context, ProposeKnowledgeRequest) (KnowledgeCandidate, error)
	ListKnowledgeCandidates(context.Context, ListKnowledgeCandidatesRequest) (KnowledgeCandidateListResponse, error)
	ReviewKnowledge(context.Context, ReviewKnowledgeRequest) (ReviewKnowledgeResponse, error)
}
