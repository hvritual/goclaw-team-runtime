package teamcontrol

import (
	"encoding/json"
	"time"
)

type CreateUserInput struct {
	ID          string
	DisplayName string
	Email       string
}

type CreateTeamInput struct {
	ID          string
	Name        string
	Description string
}

type AddTeamMemberInput struct {
	UserID string
	Role   TeamRole
}

type CreateProjectInput struct {
	ID          string
	TeamID      string
	Key         string
	Name        string
	Description string
}

type AddProjectMemberInput struct {
	UserID          string
	Role            ProjectRole
	BusinessDomains []string
	CapacityPoints  int
}

type CreateRepositoryInput struct {
	ID            string
	ProjectID     string
	Name          string
	RemoteURL     string
	LocalPath     string
	DefaultBranch string
}

type CreateIssueInput struct {
	ID                 string
	ProjectID          string
	Type               IssueType
	Title              string
	Description        string
	Severity           IssueSeverity
	Priority           IssuePriority
	Module             string
	Environment        string
	Labels             []string
	ComponentIDs       []string
	Reproduction       string
	Expected           string
	Actual             string
	ExternalIssueID    string
	DueAt              *time.Time
	SLAMinutes         int
	DuplicateOf        string
	RegressionOf       string
	IntroducedByCommit string
	FixedByCommit      string
	ReleaseID          string
}

type CreateWorkItemInput struct {
	ID                   string
	ProjectID            string
	IssueID              string
	Title                string
	Instructions         string
	BusinessDomain       string
	Priority             IssuePriority
	EstimatePoints       int
	DueAt                *time.Time
	DependsOn            []string
	ComponentIDs         []string
	VerificationCommands [][]string
}

type AssignInput struct {
	ID         string
	ProjectID  string
	TargetType AssignmentTarget
	TargetID   string
	UserID     string
	Role       AssignmentRole
}

type RegisterArtifactInput struct {
	ID           string
	ProjectID    string
	ResourceType ResourceType
	Kind         ArtifactKind
	Name         string
	URI          string
	SHA256       string
	ContentType  string
	Metadata     map[string]string
}

type CreateLinkInput struct {
	ID         string
	ProjectID  string
	SourceType ResourceType
	SourceID   string
	TargetType ResourceType
	TargetID   string
	Relation   string
	Metadata   map[string]string
}

type RegisterDocumentInput struct {
	ID         string
	ProjectID  string
	Key        string
	Title      string
	Kind       DocumentKind
	Status     DocumentStatus
	URI        string
	Revision   string
	SHA256     string
	OwnerID    string
	Supersedes string
}

type RegisterComponentInput struct {
	ID            string
	ProjectID     string
	RepositoryID  string
	Name          string
	Kind          ComponentKind
	RootPath      string
	Description   string
	OwnerIDs      []string
	DependencyIDs []string
	Metadata      map[string]string
}

type PutPolicyBundleInput struct {
	ID       string
	Name     string
	Scope    PolicyScope
	ScopeID  string
	Version  int
	Priority int
	Enabled  bool
	Rules    map[string]json.RawMessage
}

type PutTokenBudgetInput struct {
	ID          string
	ProjectID   string
	UserID      string
	LimitTokens int64
}

type RecordTokenUsageInput struct {
	ID        string
	ProjectID string
	BudgetID  string
	Tokens    int64
	TaskID    string
	Metadata  map[string]string
}

type PutKnowledgeSourceInput struct {
	ID        string
	ProjectID string
	Name      string
	URI       string
	Revision  string
	SHA256    string
	Status    RegistryStatus
	Metadata  map[string]string
}

type PutSkillReleaseInput struct {
	ID               string
	ProjectID        string
	Name             string
	Version          string
	URI              string
	SHA256           string
	MinRunnerVersion string
	Status           RegistryStatus
	Metadata         map[string]string
}

type PutRunnerReleaseInput struct {
	ID          string
	ProjectID   string
	Channel     string
	Version     string
	OS          string
	Arch        string
	URI         string
	SHA256      string
	MinProtocol string
	Status      RegistryStatus
}

type CompileContextInput struct {
	ProjectID    string
	RepositoryID string
	UserID       string
	BudgetID     string
	KnowledgeIDs []string
	SkillIDs     []string
}
