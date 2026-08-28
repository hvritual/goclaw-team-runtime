package contract

import (
	"context"
	"time"
)

type ContextCompilePolicy struct {
	Version             string
	MaxDepth            int
	MaxEntities         int
	SourceStaleAfter    time.Duration
	KnowledgeStaleAfter time.Duration
	MaxReferences       int
	MaxEstimatedTokens  int
	MaxRecentChanges    int
}

type CompileContextRequest struct {
	WorkspaceID      string
	PackID           string
	WorkItem         NodeRef
	WorkItemRevision string
	Policy           ContextCompilePolicy
}

// ContextReferenceCandidate is metadata only. The referenced content remains in
// its owning source. PublishedContextReferenceReader must return governed,
// published references; IncidentContextReferenceReader returns recorded incident
// references. Revision and checksum are mandatory for reproducible compilation.
type ContextReferenceCandidate struct {
	Kind            string
	ID              string
	Revision        string
	Checksum        string
	EntityIDs       []string
	Global          bool
	UpdatedAt       time.Time
	EstimatedTokens int
	Priority        int
}

type PublishedContextReferenceReader interface {
	ListPublishedContextReferences(ctx context.Context, workspaceID string, entityIDs []string) ([]ContextReferenceCandidate, error)
}

type IncidentContextReferenceReader interface {
	ListIncidentContextReferences(ctx context.Context, workspaceID string, entityIDs []string) ([]ContextReferenceCandidate, error)
}

type ContextSelection struct {
	Reference       ContextReference
	Source          string
	Score           int
	EstimatedTokens int
}

type ContextCompileWarning struct {
	Code     string
	EntityID string
	RefKind  string
	RefID    string
	Detail   string
}

type CompileContextResult struct {
	Pack            ContextPack
	ScopeEntityIDs  []string
	Selections      []ContextSelection
	Warnings        []ContextCompileWarning
	EstimatedTokens int
}

type ContextCompiler interface {
	Compile(ctx context.Context, request CompileContextRequest) (CompileContextResult, error)
}
