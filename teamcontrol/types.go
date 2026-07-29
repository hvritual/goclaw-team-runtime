package teamcontrol

import (
	"encoding/json"
	"time"
)

const SchemaVersion = 1

// PlannerServicePrincipal is reserved for the server-side task factory. It is
// deliberately not a User and can never authenticate or receive a token.
const PlannerServicePrincipal = "planner-service"

type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
)

type TeamRole string

const (
	TeamOwner         TeamRole = "owner"
	TeamAdmin         TeamRole = "admin"
	TeamRegularMember TeamRole = "member"
)

type ProjectRole string

const (
	ProjectOwner      ProjectRole = "owner"
	ProjectMaintainer ProjectRole = "maintainer"
	ProjectDeveloper  ProjectRole = "developer"
	ProjectReviewer   ProjectRole = "reviewer"
	ProjectViewer     ProjectRole = "viewer"
)

type MembershipStatus string

const (
	MembershipActive  MembershipStatus = "active"
	MembershipInvited MembershipStatus = "invited"
	MembershipRemoved MembershipStatus = "removed"
)

type ProjectStatus string

const (
	ProjectActive   ProjectStatus = "active"
	ProjectArchived ProjectStatus = "archived"
)

// Action is a project-scoped capability. Callers should authorize every
// resource operation with the project resolved from the stored resource, not
// from an untrusted request parameter.
type Action string

const (
	ActionProjectRead        Action = "project.read"
	ActionProjectManage      Action = "project.manage"
	ActionRepositoryRead     Action = "repository.read"
	ActionRepositoryManage   Action = "repository.manage"
	ActionIssueRead          Action = "issue.read"
	ActionIssueWrite         Action = "issue.write"
	ActionIssueAssign        Action = "issue.assign"
	ActionIssueTransition    Action = "issue.transition"
	ActionWorkItemRead       Action = "work_item.read"
	ActionWorkItemWrite      Action = "work_item.write"
	ActionWorkItemAssign     Action = "work_item.assign"
	ActionArtifactRead       Action = "artifact.read"
	ActionArtifactWrite      Action = "artifact.write"
	ActionDocumentRead       Action = "document.read"
	ActionDocumentWrite      Action = "document.write"
	ActionComponentRead      Action = "component.read"
	ActionComponentWrite     Action = "component.write"
	ActionPolicyRead         Action = "policy.read"
	ActionPolicyWrite        Action = "policy.write"
	ActionBudgetRead         Action = "budget.read"
	ActionBudgetWrite        Action = "budget.write"
	ActionKnowledgeRead      Action = "knowledge.read"
	ActionKnowledgeWrite     Action = "knowledge.write"
	ActionSkillRead          Action = "skill.read"
	ActionSkillWrite         Action = "skill.write"
	ActionRunnerReleaseRead  Action = "runner_release.read"
	ActionRunnerReleaseWrite Action = "runner_release.write"
	ActionContextRead        Action = "context.read"
	ActionContextCompile     Action = "context.compile"
)

type User struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Email       string     `json:"email,omitempty"`
	Status      UserStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AccessCredential struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Label       string     `json:"label"`
	TokenSHA256 string     `json:"token_sha256"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMembership struct {
	ID        string           `json:"id"`
	TeamID    string           `json:"team_id"`
	UserID    string           `json:"user_id"`
	Role      TeamRole         `json:"role"`
	Status    MembershipStatus `json:"status"`
	CreatedBy string           `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// TeamMember is the public domain name; TeamMembership is kept as an explicit
// alias for call sites that prefer membership terminology.
type TeamMember = TeamMembership

type Project struct {
	ID          string        `json:"id"`
	TeamID      string        `json:"team_id"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Status      ProjectStatus `json:"status"`
	CreatedBy   string        `json:"created_by"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type ProjectMembership struct {
	ID              string           `json:"id"`
	ProjectID       string           `json:"project_id"`
	UserID          string           `json:"user_id"`
	Role            ProjectRole      `json:"role"`
	Status          MembershipStatus `json:"status"`
	BusinessDomains []string         `json:"business_domains,omitempty"`
	CapacityPoints  int              `json:"capacity_points,omitempty"`
	CreatedBy       string           `json:"created_by"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type ProjectMember = ProjectMembership

type Repository struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Name          string    `json:"name"`
	RemoteURL     string    `json:"remote_url,omitempty"`
	LocalPath     string    `json:"local_path,omitempty"`
	DefaultBranch string    `json:"default_branch"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type IssueType string

const (
	IssueBug         IssueType = "bug"
	IssueTask        IssueType = "task"
	IssueImprovement IssueType = "improvement"
	IssueRisk        IssueType = "risk"
)

type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityHigh     IssueSeverity = "high"
	SeverityMedium   IssueSeverity = "medium"
	SeverityLow      IssueSeverity = "low"
)

type IssuePriority string

const (
	PriorityP0 IssuePriority = "p0"
	PriorityP1 IssuePriority = "p1"
	PriorityP2 IssuePriority = "p2"
	PriorityP3 IssuePriority = "p3"
	PriorityP4 IssuePriority = "p4"
)

type IssueStatus string

const (
	IssueNew        IssueStatus = "new"
	IssueTriaged    IssueStatus = "triaged"
	IssueAssigned   IssueStatus = "assigned"
	IssueInProgress IssueStatus = "in_progress"
	IssueBlocked    IssueStatus = "blocked"
	IssueVerifying  IssueStatus = "verifying"
	IssueResolved   IssueStatus = "resolved"
	IssueClosed     IssueStatus = "closed"
	IssueReopened   IssueStatus = "reopened"
	IssueCancelled  IssueStatus = "cancelled"
)

type Issue struct {
	ID                 string        `json:"id"`
	ProjectID          string        `json:"project_id"`
	Type               IssueType     `json:"type"`
	Title              string        `json:"title"`
	Description        string        `json:"description,omitempty"`
	Severity           IssueSeverity `json:"severity"`
	Priority           IssuePriority `json:"priority"`
	Status             IssueStatus   `json:"status"`
	ReporterID         string        `json:"reporter_id"`
	Module             string        `json:"module,omitempty"`
	Environment        string        `json:"environment,omitempty"`
	Labels             []string      `json:"labels,omitempty"`
	ComponentIDs       []string      `json:"component_ids,omitempty"`
	Reproduction       string        `json:"reproduction,omitempty"`
	Expected           string        `json:"expected,omitempty"`
	Actual             string        `json:"actual,omitempty"`
	ExternalIssueID    string        `json:"external_issue_id,omitempty"`
	DueAt              *time.Time    `json:"due_at,omitempty"`
	SLAMinutes         int           `json:"sla_minutes,omitempty"`
	SLADeadline        *time.Time    `json:"sla_deadline,omitempty"`
	DuplicateOf        string        `json:"duplicate_of,omitempty"`
	RegressionOf       string        `json:"regression_of,omitempty"`
	IntroducedByCommit string        `json:"introduced_by_commit,omitempty"`
	FixedByCommit      string        `json:"fixed_by_commit,omitempty"`
	ReleaseID          string        `json:"release_id,omitempty"`
	ReopenCount        int           `json:"reopen_count"`
	Resolution         string        `json:"resolution,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type WorkItemStatus string

const (
	WorkItemPending    WorkItemStatus = "pending"
	WorkItemReady      WorkItemStatus = "ready"
	WorkItemInProgress WorkItemStatus = "in_progress"
	WorkItemBlocked    WorkItemStatus = "blocked"
	WorkItemVerifying  WorkItemStatus = "verifying"
	WorkItemDone       WorkItemStatus = "done"
	WorkItemCancelled  WorkItemStatus = "cancelled"
)

type WorkItem struct {
	ID                   string         `json:"id"`
	ProjectID            string         `json:"project_id"`
	IssueID              string         `json:"issue_id,omitempty"`
	Title                string         `json:"title"`
	Instructions         string         `json:"instructions"`
	BusinessDomain       string         `json:"business_domain,omitempty"`
	Priority             IssuePriority  `json:"priority"`
	EstimatePoints       int            `json:"estimate_points,omitempty"`
	DueAt                *time.Time     `json:"due_at,omitempty"`
	Status               WorkItemStatus `json:"status"`
	DependsOn            []string       `json:"depends_on,omitempty"`
	ComponentIDs         []string       `json:"component_ids,omitempty"`
	VerificationCommands [][]string     `json:"verification_commands,omitempty"`
	CreatedBy            string         `json:"created_by"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type AssignmentTarget string

const (
	AssignmentIssue    AssignmentTarget = "issue"
	AssignmentWorkItem AssignmentTarget = "work_item"
)

type AssignmentRole string

const (
	AssignmentOwner       AssignmentRole = "owner"
	AssignmentContributor AssignmentRole = "contributor"
	AssignmentReviewer    AssignmentRole = "reviewer"
)

type AssignmentStatus string

const (
	AssignmentActive   AssignmentStatus = "active"
	AssignmentReleased AssignmentStatus = "released"
)

type Assignment struct {
	ID         string           `json:"id"`
	ProjectID  string           `json:"project_id"`
	TargetType AssignmentTarget `json:"target_type"`
	TargetID   string           `json:"target_id"`
	UserID     string           `json:"user_id"`
	Role       AssignmentRole   `json:"role"`
	Status     AssignmentStatus `json:"status"`
	AssignedBy string           `json:"assigned_by"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type ArtifactKind string

const (
	ArtifactDiff     ArtifactKind = "diff"
	ArtifactEvidence ArtifactKind = "evidence"
	ArtifactBuild    ArtifactKind = "build"
	ArtifactLog      ArtifactKind = "log"
	ArtifactReport   ArtifactKind = "report"
	ArtifactTrace    ArtifactKind = "trace"
	ArtifactCommit   ArtifactKind = "commit"
	ArtifactPR       ArtifactKind = "pull_request"
	ArtifactPackage  ArtifactKind = "package"
	ArtifactOther    ArtifactKind = "other"
)

type Artifact struct {
	ID           string            `json:"id"`
	ProjectID    string            `json:"project_id"`
	ResourceType ResourceType      `json:"resource_type"`
	Kind         ArtifactKind      `json:"kind"`
	Name         string            `json:"name"`
	URI          string            `json:"uri"`
	SHA256       string            `json:"sha256,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedBy    string            `json:"created_by"`
	CreatedAt    time.Time         `json:"created_at"`
}

type ResourceType string

const (
	ResourceIssue          ResourceType = "issue"
	ResourceTask           ResourceType = "task"
	ResourceWorkItem       ResourceType = "work_item"
	ResourceRun            ResourceType = "run"
	ResourceTrace          ResourceType = "trace"
	ResourceCommit         ResourceType = "commit"
	ResourcePullRequest    ResourceType = "pull_request"
	ResourceCI             ResourceType = "ci"
	ResourceRelease        ResourceType = "release"
	ResourceRegressionCase ResourceType = "regression_case"
	ResourceSpec           ResourceType = "spec"
	ResourceArtifact       ResourceType = "artifact"
	ResourceDocument       ResourceType = "document"
	ResourceComponent      ResourceType = "component"
	ResourceRepository     ResourceType = "repository"
	ResourcePolicy         ResourceType = "policy"
)

type CorrelationLink struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	SourceType ResourceType      `json:"source_type"`
	SourceID   string            `json:"source_id"`
	TargetType ResourceType      `json:"target_type"`
	TargetID   string            `json:"target_id"`
	Relation   string            `json:"relation"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedBy  string            `json:"created_by"`
	CreatedAt  time.Time         `json:"created_at"`
}

type DocumentKind string

const (
	DocumentPRD       DocumentKind = "prd"
	DocumentADR       DocumentKind = "adr"
	DocumentDesign    DocumentKind = "design"
	DocumentRunbook   DocumentKind = "runbook"
	DocumentAPI       DocumentKind = "api"
	DocumentTestPlan  DocumentKind = "test_plan"
	DocumentReport    DocumentKind = "report"
	DocumentKnowledge DocumentKind = "knowledge"
	DocumentOther     DocumentKind = "other"
)

type DocumentStatus string

const (
	DocumentDraft      DocumentStatus = "draft"
	DocumentActive     DocumentStatus = "active"
	DocumentSuperseded DocumentStatus = "superseded"
	DocumentArchived   DocumentStatus = "archived"
)

type Document struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"project_id"`
	Key        string         `json:"key"`
	Title      string         `json:"title"`
	Kind       DocumentKind   `json:"kind"`
	Status     DocumentStatus `json:"status"`
	URI        string         `json:"uri"`
	Revision   string         `json:"revision,omitempty"`
	SHA256     string         `json:"sha256,omitempty"`
	OwnerID    string         `json:"owner_id,omitempty"`
	Supersedes string         `json:"supersedes,omitempty"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type ComponentKind string

const (
	ComponentService ComponentKind = "service"
	ComponentLibrary ComponentKind = "library"
	ComponentApp     ComponentKind = "app"
	ComponentModule  ComponentKind = "module"
	ComponentDevice  ComponentKind = "device"
	ComponentOther   ComponentKind = "other"
)

type Component struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"project_id"`
	RepositoryID  string            `json:"repository_id,omitempty"`
	Name          string            `json:"name"`
	Kind          ComponentKind     `json:"kind"`
	RootPath      string            `json:"root_path,omitempty"`
	Description   string            `json:"description,omitempty"`
	OwnerIDs      []string          `json:"owner_ids,omitempty"`
	DependencyIDs []string          `json:"dependency_ids,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedBy     string            `json:"created_by"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type PolicyScope string

const (
	PolicyTeam       PolicyScope = "team"
	PolicyProject    PolicyScope = "project"
	PolicyRepository PolicyScope = "repository"
	PolicyComponent  PolicyScope = "component"
)

// PolicyBundle is a versioned layer. Rules are canonical JSON values. During
// resolution, layers are applied team -> project -> repository -> component;
// later layers override keys from earlier layers.
type PolicyBundle struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Scope     PolicyScope                `json:"scope"`
	ScopeID   string                     `json:"scope_id"`
	TeamID    string                     `json:"team_id"`
	ProjectID string                     `json:"project_id,omitempty"`
	Version   int                        `json:"version"`
	Priority  int                        `json:"priority"`
	Enabled   bool                       `json:"enabled"`
	Rules     map[string]json.RawMessage `json:"rules"`
	Hash      string                     `json:"hash"`
	CreatedBy string                     `json:"created_by"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

type ResolvedPolicy struct {
	ProjectID    string                     `json:"project_id"`
	RepositoryID string                     `json:"repository_id,omitempty"`
	ComponentID  string                     `json:"component_id,omitempty"`
	BundleIDs    []string                   `json:"bundle_ids"`
	BundleHashes []string                   `json:"bundle_hashes"`
	Rules        map[string]json.RawMessage `json:"rules"`
	Hash         string                     `json:"hash"`
}

type RegistryStatus string

const (
	RegistryDraft    RegistryStatus = "draft"
	RegistryApproved RegistryStatus = "approved"
	RegistryDisabled RegistryStatus = "disabled"
)

type TokenBudget struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	UserID      string    `json:"user_id,omitempty"`
	LimitTokens int64     `json:"limit_tokens"`
	UsedTokens  int64     `json:"used_tokens"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TokenUsageEvent struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	BudgetID   string            `json:"budget_id"`
	Tokens     int64             `json:"tokens"`
	TaskID     string            `json:"task_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	RecordedBy string            `json:"recorded_by"`
	RecordedAt time.Time         `json:"recorded_at"`
}

type KnowledgeSource struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Name      string            `json:"name"`
	URI       string            `json:"uri"`
	Revision  string            `json:"revision"`
	SHA256    string            `json:"sha256"`
	Status    RegistryStatus    `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedBy string            `json:"created_by"`
	UpdatedBy string            `json:"updated_by"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type SkillRelease struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	URI              string            `json:"uri"`
	SHA256           string            `json:"sha256"`
	MinRunnerVersion string            `json:"min_runner_version,omitempty"`
	Status           RegistryStatus    `json:"status"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedBy        string            `json:"created_by"`
	UpdatedBy        string            `json:"updated_by"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type RunnerRelease struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	Channel     string         `json:"channel"`
	Version     string         `json:"version"`
	OS          string         `json:"os"`
	Arch        string         `json:"arch"`
	URI         string         `json:"uri"`
	SHA256      string         `json:"sha256"`
	MinProtocol string         `json:"min_protocol"`
	Status      RegistryStatus `json:"status"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ContextResourceRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	URI     string `json:"uri"`
	SHA256  string `json:"sha256"`
}

type ContextBudgetSnapshot struct {
	BudgetID     string `json:"budget_id,omitempty"`
	BudgetUserID string `json:"budget_user_id,omitempty"`
	LegacyUserID string `json:"user_id,omitempty"`
	LimitTokens  int64  `json:"limit_tokens,omitempty"`
	UsedTokens   int64  `json:"used_tokens,omitempty"`
}

type ContextBundle struct {
	ID              string                `json:"id"`
	ProjectID       string                `json:"project_id"`
	RepositoryID    string                `json:"repository_id,omitempty"`
	TargetUserID    string                `json:"target_user_id,omitempty"`
	CompilerVersion string                `json:"compiler_version"`
	Policy          ResolvedPolicy        `json:"policy"`
	Budget          ContextBudgetSnapshot `json:"budget"`
	Knowledge       []ContextResourceRef  `json:"knowledge"`
	Skills          []ContextResourceRef  `json:"skills"`
	Hash            string                `json:"hash"`
	CreatedBy       string                `json:"created_by"`
	CreatedAt       time.Time             `json:"created_at"`
}

type state struct {
	SchemaVersion      int                          `json:"schema_version"`
	Revision           uint64                       `json:"revision"`
	Users              map[string]User              `json:"users"`
	AccessCredentials  map[string]AccessCredential  `json:"access_credentials"`
	Teams              map[string]Team              `json:"teams"`
	TeamMemberships    map[string]TeamMembership    `json:"team_memberships"`
	Projects           map[string]Project           `json:"projects"`
	ProjectMemberships map[string]ProjectMembership `json:"project_memberships"`
	Repositories       map[string]Repository        `json:"repositories"`
	Issues             map[string]Issue             `json:"issues"`
	WorkItems          map[string]WorkItem          `json:"work_items"`
	Assignments        map[string]Assignment        `json:"assignments"`
	Artifacts          map[string]Artifact          `json:"artifacts"`
	Links              map[string]CorrelationLink   `json:"links"`
	Documents          map[string]Document          `json:"documents"`
	Components         map[string]Component         `json:"components"`
	Policies           map[string]PolicyBundle      `json:"policies"`
	TokenBudgets       map[string]TokenBudget       `json:"token_budgets"`
	TokenUsageEvents   map[string]TokenUsageEvent   `json:"token_usage_events"`
	KnowledgeSources   map[string]KnowledgeSource   `json:"knowledge_sources"`
	SkillReleases      map[string]SkillRelease      `json:"skill_releases"`
	RunnerReleases     map[string]RunnerRelease     `json:"runner_releases"`
	ContextBundles     map[string]ContextBundle     `json:"context_bundles"`
	UpdatedAt          time.Time                    `json:"updated_at"`
}

func newState() state {
	return state{
		SchemaVersion:      SchemaVersion,
		Users:              make(map[string]User),
		AccessCredentials:  make(map[string]AccessCredential),
		Teams:              make(map[string]Team),
		TeamMemberships:    make(map[string]TeamMembership),
		Projects:           make(map[string]Project),
		ProjectMemberships: make(map[string]ProjectMembership),
		Repositories:       make(map[string]Repository),
		Issues:             make(map[string]Issue),
		WorkItems:          make(map[string]WorkItem),
		Assignments:        make(map[string]Assignment),
		Artifacts:          make(map[string]Artifact),
		Links:              make(map[string]CorrelationLink),
		Documents:          make(map[string]Document),
		Components:         make(map[string]Component),
		Policies:           make(map[string]PolicyBundle),
		TokenBudgets:       make(map[string]TokenBudget),
		TokenUsageEvents:   make(map[string]TokenUsageEvent),
		KnowledgeSources:   make(map[string]KnowledgeSource),
		SkillReleases:      make(map[string]SkillRelease),
		RunnerReleases:     make(map[string]RunnerRelease),
		ContextBundles:     make(map[string]ContextBundle),
		UpdatedAt:          time.Now().UTC(),
	}
}
