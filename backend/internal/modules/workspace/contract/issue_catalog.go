package contract

import (
	"context"
	"encoding/json"
)

type IssueLabel struct {
	ID           string
	WorkspaceID  string
	ResourceType string
	Name         string
	Description  string
	Color        string
	UsageCount   int64
	CreatedAt    string
	UpdatedAt    string
}

type CreateIssueLabelRequest struct {
	WorkspaceID  string
	ResourceType string
	Name         string
	Description  string
	Color        string
}

type UpdateIssueLabelRequest struct {
	WorkspaceID string
	LabelID     string
	Name        *string
	Description *string
	Color       *string
}

type IssuePropertyOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type IssuePropertyConfig struct {
	Options []IssuePropertyOption `json:"options,omitempty"`
}

type IssuePropertyDefinition struct {
	ID          string
	WorkspaceID string
	Name        string
	Type        string
	Description string
	Icon        string
	Config      IssuePropertyConfig
	Position    float64
	Archived    bool
	ArchivedAt  *string
	UsageCount  int64
	CreatedAt   string
	UpdatedAt   string
}

type CreateIssuePropertyRequest struct {
	WorkspaceID string
	Name        string
	Type        string
	Description string
	Icon        string
	Config      *IssuePropertyConfig
}

type UpdateIssuePropertyRequest struct {
	WorkspaceID string
	PropertyID  string
	Name        *string
	Description *string
	Icon        *string
	Config      *IssuePropertyConfig
	Archived    *bool
}

type AcceptanceConclusionInput struct {
	Result       string
	Rationale    string
	EvidenceRefs []string
}

type AcceptanceConclusion struct {
	ID           string
	WorkspaceID  string
	IssueID      string
	Result       string
	Rationale    string
	EvidenceRefs []string
	ActorID      string
	CreatedAt    string
	UpdatedAt    string
}

type AcceptanceConclusionMutation struct {
	Conclusion AcceptanceConclusion
	Issue      *Issue
}

type IssueCatalogService interface {
	ListIssueLabels(context.Context, string) ([]IssueLabel, error)
	GetIssueLabel(context.Context, string, string) (IssueLabel, error)
	CreateIssueLabel(context.Context, CreateIssueLabelRequest) (IssueLabel, error)
	UpdateIssueLabel(context.Context, UpdateIssueLabelRequest) (IssueLabel, error)
	DeleteIssueLabel(context.Context, string, string) error
	ListLabelsForIssue(context.Context, string, string) (string, []IssueLabel, error)
	AttachLabelToIssue(context.Context, string, string, string) (string, []IssueLabel, error)
	DetachLabelFromIssue(context.Context, string, string, string) (string, []IssueLabel, error)

	ListIssueProperties(context.Context, string, bool) ([]IssuePropertyDefinition, error)
	GetIssueProperty(context.Context, string, string) (IssuePropertyDefinition, error)
	CreateIssueProperty(context.Context, CreateIssuePropertyRequest) (IssuePropertyDefinition, error)
	UpdateIssueProperty(context.Context, UpdateIssuePropertyRequest) (IssuePropertyDefinition, error)
	SetIssueProperty(context.Context, string, string, string, json.RawMessage) (string, map[string]any, error)
	UnsetIssueProperty(context.Context, string, string, string) (string, map[string]any, error)

	ListAcceptanceConclusions(context.Context, string, string) (string, []AcceptanceConclusion, error)
	CreateAcceptanceConclusion(context.Context, string, string, AcceptanceConclusionInput) (AcceptanceConclusionMutation, error)
	CompleteIssueWithAcceptance(context.Context, string, string, AcceptanceConclusionInput) (AcceptanceConclusionMutation, error)
}
