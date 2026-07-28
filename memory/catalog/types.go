package catalog

import (
	"time"

	"github.com/smallnest/goclaw/governance"
)

const SchemaVersion = 1

type RecordKind string

const (
	KindGoal         RecordKind = "goal"
	KindDecision     RecordKind = "decision"
	KindConstraint   RecordKind = "constraint"
	KindRequirement  RecordKind = "requirement"
	KindFact         RecordKind = "fact"
	KindPreference   RecordKind = "preference"
	KindProcedure    RecordKind = "procedure"
	KindLesson       RecordKind = "lesson"
	KindContext      RecordKind = "context"
	KindConversation RecordKind = "conversation"
	KindSource       RecordKind = "source"
)

type RecordStatus string

const (
	StatusPending     RecordStatus = "pending"
	StatusActive      RecordStatus = "active"
	StatusRejected    RecordStatus = "rejected"
	StatusSuperseded  RecordStatus = "superseded"
	StatusWithdrawn   RecordStatus = "withdrawn"
	StatusQuarantined RecordStatus = "quarantined"
)

type RelationType string

const (
	RelationSupersedes  RelationType = "supersedes"
	RelationContradicts RelationType = "contradicts"
	RelationDerivedFrom RelationType = "derived_from"
	RelationSupports    RelationType = "supports"
	RelationRelatedTo   RelationType = "related_to"
)

type AuthorityType string

const (
	AuthorityPerson       AuthorityType = "person"
	AuthorityOrganization AuthorityType = "organization"
	AuthorityProject      AuthorityType = "project"
	AuthoritySystem       AuthorityType = "system"
	AuthorityTopic        AuthorityType = "topic"
	AuthorityPlace        AuthorityType = "place"
	AuthorityDevice       AuthorityType = "device"
)

type AuthorityStatus string

const (
	AuthorityActive     AuthorityStatus = "active"
	AuthorityDeprecated AuthorityStatus = "deprecated"
	AuthorityRedirected AuthorityStatus = "redirected"
)

// Config controls the catalog layer. The catalog is the governed control plane;
// vector and QMD indexes remain optional content-level retrieval backends.
type Config struct {
	Enabled           bool     `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	DatabasePath      string   `mapstructure:"database_path" json:"database_path" yaml:"database_path"`
	DefaultProject    string   `mapstructure:"default_project" json:"default_project" yaml:"default_project"`
	ReviewAfterDays   int      `mapstructure:"review_after_days" json:"review_after_days" yaml:"review_after_days"`
	MaxContextRecords int      `mapstructure:"max_context_records" json:"max_context_records" yaml:"max_context_records"`
	MaxContextChars   int      `mapstructure:"max_context_chars" json:"max_context_chars" yaml:"max_context_chars"`
	AutoIngest        bool     `mapstructure:"auto_ingest" json:"auto_ingest" yaml:"auto_ingest"`
	SourceRoot        string   `mapstructure:"source_root" json:"source_root,omitempty" yaml:"source_root,omitempty"`
	SourcePaths       []string `mapstructure:"source_paths" json:"source_paths,omitempty" yaml:"source_paths,omitempty"`
	SourceScheme      string   `mapstructure:"source_scheme" json:"source_scheme,omitempty" yaml:"source_scheme,omitempty"`
	SourceKind        string   `mapstructure:"source_kind" json:"source_kind,omitempty" yaml:"source_kind,omitempty"`
	SourceRevision    string   `mapstructure:"source_revision" json:"source_revision,omitempty" yaml:"source_revision,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		DefaultProject:    "default",
		ReviewAfterDays:   90,
		MaxContextRecords: 6,
		MaxContextChars:   8000,
		SourceScheme:      "markdown",
		SourceKind:        "markdown",
	}
}

// Relation makes cross-record meaning explicit instead of relying on textual
// similarity. TargetID always identifies another catalog record.
type Relation struct {
	Type     RelationType `json:"type" yaml:"type"`
	TargetID string       `json:"target_id" yaml:"target_id"`
	Note     string       `json:"note,omitempty" yaml:"note,omitempty"`
}

// Provenance follows the PROV Entity/Activity/Agent shape without requiring an
// RDF store. SourceURI should be stable across synchronized computers.
type Provenance struct {
	SourceURI      string    `json:"source_uri" yaml:"source_uri"`
	SourceKind     string    `json:"source_kind,omitempty" yaml:"source_kind,omitempty"`
	SourceRevision string    `json:"source_revision,omitempty" yaml:"source_revision,omitempty"`
	SourceSHA256   string    `json:"source_sha256,omitempty" yaml:"source_sha256,omitempty"`
	CapturedAt     time.Time `json:"captured_at" yaml:"captured_at"`
	AgentID        string    `json:"agent_id,omitempty" yaml:"agent_id,omitempty"`
	ActivityID     string    `json:"activity_id,omitempty" yaml:"activity_id,omitempty"`
	TraceID        string    `json:"trace_id,omitempty" yaml:"trace_id,omitempty"`
}

// Record is a catalog description plus an immutable content manifestation.
// WorkID groups one durable concept; ExpressionID groups one language/form;
// ManifestationID identifies this version; ItemID identifies its source copy.
type Record struct {
	SchemaVersion   int                        `json:"schema_version" yaml:"schema_version"`
	ID              string                     `json:"id" yaml:"id"`
	ProjectID       string                     `json:"project_id" yaml:"project_id"`
	Collection      string                     `json:"collection" yaml:"collection"`
	WorkID          string                     `json:"work_id" yaml:"work_id"`
	ExpressionID    string                     `json:"expression_id" yaml:"expression_id"`
	ManifestationID string                     `json:"manifestation_id" yaml:"manifestation_id"`
	ItemID          string                     `json:"item_id" yaml:"item_id"`
	Title           string                     `json:"title" yaml:"title"`
	Abstract        string                     `json:"abstract,omitempty" yaml:"abstract,omitempty"`
	Content         string                     `json:"content" yaml:"content"`
	Kind            RecordKind                 `json:"kind" yaml:"kind"`
	Status          RecordStatus               `json:"status" yaml:"status"`
	Language        string                     `json:"language,omitempty" yaml:"language,omitempty"`
	Subjects        []string                   `json:"subjects,omitempty" yaml:"subjects,omitempty"`
	Facets          map[string][]string        `json:"facets,omitempty" yaml:"facets,omitempty"`
	AuthorityIDs    []string                   `json:"authority_ids,omitempty" yaml:"authority_ids,omitempty"`
	Relations       []Relation                 `json:"relations,omitempty" yaml:"relations,omitempty"`
	Provenance      Provenance                 `json:"provenance" yaml:"provenance"`
	EvidenceRefs    []string                   `json:"evidence_refs,omitempty" yaml:"evidence_refs,omitempty"`
	Confidence      float64                    `json:"confidence" yaml:"confidence"`
	ValidFrom       *time.Time                 `json:"valid_from,omitempty" yaml:"valid_from,omitempty"`
	ValidUntil      *time.Time                 `json:"valid_until,omitempty" yaml:"valid_until,omitempty"`
	ReviewAt        *time.Time                 `json:"review_at,omitempty" yaml:"review_at,omitempty"`
	ExpiresAt       *time.Time                 `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	Version         int                        `json:"version" yaml:"version"`
	Checksum        string                     `json:"checksum" yaml:"checksum"`
	CreatedBy       string                     `json:"created_by" yaml:"created_by"`
	CreatedAt       time.Time                  `json:"created_at" yaml:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at" yaml:"updated_at"`
	ReviewedBy      string                     `json:"reviewed_by,omitempty" yaml:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time                 `json:"reviewed_at,omitempty" yaml:"reviewed_at,omitempty"`
	Decision        *governance.DecisionRecord `json:"decision,omitempty" yaml:"decision,omitempty"`
}

type CreateInput struct {
	ProjectID    string
	Collection   string
	WorkID       string
	ExpressionID string
	Title        string
	Abstract     string
	Content      string
	Kind         RecordKind
	Language     string
	Subjects     []string
	Facets       map[string][]string
	AuthorityIDs []string
	Relations    []Relation
	Provenance   Provenance
	EvidenceRefs []string
	Confidence   float64
	ValidFrom    *time.Time
	ValidUntil   *time.Time
	ReviewAt     *time.Time
	ExpiresAt    *time.Time
	CreatedBy    string
}

type SearchQuery struct {
	Query          string
	ProjectID      string
	Statuses       []RecordStatus
	Kinds          []RecordKind
	Facets         map[string][]string
	AuthorityIDs   []string
	IncludeShared  bool
	IncludeExpired bool
	Limit          int
}

type SearchResult struct {
	Record    Record   `json:"record"`
	Score     float64  `json:"score"`
	MatchedBy []string `json:"matched_by,omitempty"`
	Citation  string   `json:"citation"`
	Warnings  []string `json:"warnings,omitempty"`
	ReviewDue bool     `json:"review_due"`
	Expired   bool     `json:"expired"`
}

type Authority struct {
	SchemaVersion  int                        `json:"schema_version"`
	ID             string                     `json:"id"`
	ProjectID      string                     `json:"project_id"`
	Type           AuthorityType              `json:"type"`
	PreferredLabel string                     `json:"preferred_label"`
	Aliases        []string                   `json:"aliases,omitempty"`
	Description    string                     `json:"description,omitempty"`
	ExternalIDs    map[string]string          `json:"external_ids,omitempty"`
	Status         AuthorityStatus            `json:"status"`
	RedirectTo     string                     `json:"redirect_to,omitempty"`
	CreatedBy      string                     `json:"created_by"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
	Decision       *governance.DecisionRecord `json:"decision,omitempty"`
}

type AuthorityInput struct {
	ID             string
	ProjectID      string
	Type           AuthorityType
	PreferredLabel string
	Aliases        []string
	Description    string
	ExternalIDs    map[string]string
	CreatedBy      string
}

type UsageKind string

const (
	UsageRetrieved UsageKind = "retrieved"
	UsageCited     UsageKind = "cited"
	UsageAccepted  UsageKind = "accepted"
	UsageRejected  UsageKind = "rejected"
)

type CirculationEvent struct {
	ID        string            `json:"id"`
	RecordID  string            `json:"record_id"`
	ProjectID string            `json:"project_id"`
	Kind      UsageKind         `json:"kind"`
	Actor     string            `json:"actor,omitempty"`
	TraceID   string            `json:"trace_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type Stats struct {
	TotalRecords             int                  `json:"total_records"`
	ByStatus                 map[RecordStatus]int `json:"by_status"`
	ReviewDue                int                  `json:"review_due"`
	Expired                  int                  `json:"expired"`
	Authorities              int                  `json:"authorities"`
	UnresolvedContradictions int                  `json:"unresolved_contradictions"`
	UsageLast30Days          int                  `json:"usage_last_30_days"`
}

type IngestOptions struct {
	ProjectID      string
	Collection     string
	DefaultKind    RecordKind
	SourceRoot     string
	SourceScheme   string
	SourceKind     string
	SourceRevision string
	Actor          string
}

type IngestReport struct {
	Scanned  int      `json:"scanned"`
	Created  int      `json:"created"`
	Existing int      `json:"existing"`
	Failed   int      `json:"failed"`
	Records  []string `json:"records,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}
