package knowledge

import "time"

type StoreCapabilities struct {
	SchemaVersion int
	JournalMode   string
	ForeignKeys   bool
	FTS5          bool
}

type Kind string

const (
	KindGoal        Kind = "goal"
	KindDecision    Kind = "decision"
	KindConstraint  Kind = "constraint"
	KindRequirement Kind = "requirement"
	KindProcedure   Kind = "procedure"
	KindLesson      Kind = "lesson"
	KindReference   Kind = "reference"
)

type Status string

const (
	StatusCandidate   Status = "candidate"
	StatusInReview    Status = "in_review"
	StatusPublished   Status = "published"
	StatusSuperseded  Status = "superseded"
	StatusRejected    Status = "rejected"
	StatusQuarantined Status = "quarantined"
)

type ProposalInput struct {
	WorkspaceID string
	ProjectID   string
	Kind        Kind
	Title       string
	Content     string
	Reason      string
	ProposedBy  string
	SourceRefs  []SourceRef
}

type Candidate struct {
	ID          string
	WorkspaceID string
	ProjectID   string
	Kind        Kind
	Title       string
	Content     string
	Reason      string
	Status      Status
	Revision    int64
	ProposedBy  string
	SourceRefs  []SourceRef
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SourceRef struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Revision string `json:"revision"`
	URI      string `json:"uri"`
	Checksum string `json:"checksum"`
}

type ReviewAction string

const (
	ReviewApprove    ReviewAction = "approve"
	ReviewReject     ReviewAction = "reject"
	ReviewQuarantine ReviewAction = "quarantine"
)

type ReviewInput struct {
	WorkspaceID      string
	CandidateID      string
	ExpectedRevision int64
	Action           ReviewAction
	ReviewerID       string
	Rationale        string
}

type Review struct {
	Action      ReviewAction
	ReviewerID  string
	Rationale   string
	ReviewedAt  time.Time
	OldRevision int64
	NewRevision int64
}

type Revision struct {
	Number     int64       `json:"number"`
	Title      string      `json:"title"`
	Content    string      `json:"content"`
	CreatedBy  string      `json:"created_by"`
	CreatedAt  time.Time   `json:"created_at"`
	SourceRefs []SourceRef `json:"source_refs"`
}

type Entry struct {
	ID              string
	WorkspaceID     string
	ProjectID       string
	CandidateID     string
	Kind            Kind
	Status          Status
	CurrentRevision int64
	Revisions       []Revision
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ReviewCommand struct {
	CandidateID      string
	WorkspaceID      string
	ExpectedRevision int64
	NewStatus        Status
	Review           Review
	Entry            *Entry
}

type Evidence struct {
	ID             string
	WorkspaceID    string
	ProjectID      string
	SourceType     string
	SourceID       string
	SourceRevision string
	EventType      string
	Kind           Kind
	Title          string
	Content        string
	ActorID        string
	IdempotencyKey string
	ProvenanceURI  string
	Checksum       string
	OccurredAt     time.Time
	Terminal       bool
	Validated      bool
	HasConflict    bool
	Confidence     float64
	SourceRefs     []SourceRef
	Metadata       map[string]string
}

type PromotionAction string

const (
	PromotionCandidate  PromotionAction = "candidate"
	PromotionPublish    PromotionAction = "publish"
	PromotionQuarantine PromotionAction = "quarantine"
)

type PromotionDecision struct {
	Action PromotionAction
	Reason string
}

type IngestionCommand struct {
	Evidence  Evidence
	Candidate *Candidate
	Entry     *Entry
}

type IngestionResult struct {
	Duplicate bool
	Candidate *Candidate
	Entry     *Entry
}

type SearchQuery struct {
	WorkspaceID string
	ProjectID   string
	Kinds       []Kind
	Text        string
	Limit       int
	Cursor      string
}

type SearchResult struct {
	Entry     Entry
	Score     float64
	MatchedBy []string
	Citation  string
}

type SearchPage struct {
	Results    []SearchResult
	NextCursor string
}

type CandidateQuery struct {
	WorkspaceID string
	ProjectID   string
	Statuses    []Status
	Kinds       []Kind
	Limit       int
	Cursor      string
}

type CandidatePage struct {
	Candidates []Candidate
	NextCursor string
}
