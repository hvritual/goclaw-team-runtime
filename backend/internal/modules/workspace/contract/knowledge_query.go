package contract

import (
	"context"
	"errors"
)

var (
	ErrInvalidKnowledgeQuery = errors.New("invalid Knowledge query")
	ErrKnowledgeQueryHidden  = errors.New("Knowledge not found")
)

type KnowledgeSourceRef struct {
	Type           string  `json:"type"`
	ID             string  `json:"id"`
	Revision       string  `json:"revision"`
	Citation       string  `json:"citation"`
	AssetID        *string `json:"asset_id"`
	AssetVersionID *string `json:"asset_version_id"`
}

type KnowledgeRevision struct {
	Number             int                  `json:"number"`
	SupersedesRevision int                  `json:"supersedes_revision"`
	Title              string               `json:"title"`
	Content            string               `json:"content"`
	CreatedBy          string               `json:"created_by"`
	CreatedAt          string               `json:"created_at"`
	SourceRefs         []KnowledgeSourceRef `json:"source_refs"`
}

type GovernedKnowledgeEntry struct {
	ID              string              `json:"id"`
	WorkspaceID     string              `json:"workspace_id"`
	ProjectID       *string             `json:"project_id"`
	CandidateID     *string             `json:"candidate_id"`
	Kind            string              `json:"kind"`
	Status          string              `json:"status"`
	CurrentRevision int                 `json:"current_revision"`
	Revision        KnowledgeRevision   `json:"revision"`
	Revisions       []KnowledgeRevision `json:"revisions,omitempty"`
	Citation        string              `json:"citation"`
	MatchedBy       string              `json:"matched_by"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
}

type QueryKnowledgeRequest struct {
	WorkspaceID    string
	Query          string
	Statuses       []string
	Kinds          []string
	SourceType     string
	SourceID       string
	SourceRevision string
	Applicability  string
	ProjectID      string
	Revision       int
	Limit          int
	Cursor         string
}

type QueryKnowledgeResponse struct {
	Entries    []GovernedKnowledgeEntry `json:"entries"`
	Total      int                      `json:"total"`
	NextCursor *string                  `json:"next_cursor"`
}

type KnowledgeQueryService interface {
	QueryKnowledge(context.Context, QueryKnowledgeRequest) (QueryKnowledgeResponse, error)
	GetGovernedKnowledge(context.Context, string, string) (GovernedKnowledgeEntry, error)
}
