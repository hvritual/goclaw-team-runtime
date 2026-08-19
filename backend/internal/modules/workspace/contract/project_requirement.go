package contract

import "context"

type ProjectRequirementItem struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type ProjectRequirementContent struct {
	ProblemStatement   string                   `json:"problem_statement"`
	Goals              []ProjectRequirementItem `json:"goals"`
	InScope            []ProjectRequirementItem `json:"in_scope"`
	OutOfScope         []ProjectRequirementItem `json:"out_of_scope"`
	Constraints        []ProjectRequirementItem `json:"constraints"`
	AcceptanceCriteria []ProjectRequirementItem `json:"acceptance_criteria"`
	Dependencies       []ProjectRequirementItem `json:"dependencies"`
}

type ProjectRequirementBaseline struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspace_id"`
	ProjectID         string  `json:"project_id"`
	Status            string  `json:"status"`
	CurrentRevision   int64   `json:"current_revision"`
	ApprovedRevision  *int64  `json:"approved_revision"`
	EffectiveRevision *int64  `json:"effective_revision"`
	SubmittedBy       *string `json:"submitted_by"`
	SubmittedAt       *string `json:"submitted_at"`
	ApprovedBy        *string `json:"approved_by"`
	ApprovedAt        *string `json:"approved_at"`
	FrozenBy          *string `json:"frozen_by"`
	FrozenAt          *string `json:"frozen_at"`
	RetiredBy         *string `json:"retired_by"`
	RetiredAt         *string `json:"retired_at"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type ProjectRequirementRevision struct {
	BaselineID    string                    `json:"baseline_id"`
	Revision      int64                     `json:"revision"`
	Content       ProjectRequirementContent `json:"content"`
	State         string                    `json:"state"`
	Action        string                    `json:"action"`
	ChangeSummary string                    `json:"change_summary"`
	ActorID       string                    `json:"actor_id"`
	SubmittedBy   *string                   `json:"submitted_by"`
	SubmittedAt   *string                   `json:"submitted_at"`
	ApprovedBy    *string                   `json:"approved_by"`
	ApprovedAt    *string                   `json:"approved_at"`
	FrozenBy      *string                   `json:"frozen_by"`
	FrozenAt      *string                   `json:"frozen_at"`
	CreatedAt     string                    `json:"created_at"`
}

type ProjectRequirementIssueLink struct {
	RequirementKey string  `json:"requirement_key"`
	IssueID        string  `json:"issue_id"`
	Identifier     string  `json:"identifier"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	LinkedRevision int64   `json:"linked_revision"`
	ReviewRequired bool    `json:"review_required"`
	LinkedBy       string  `json:"linked_by"`
	LinkedAt       string  `json:"linked_at"`
	UnlinkedAt     *string `json:"unlinked_at"`
}

type ProjectRequirementOutlineLink struct {
	RequirementKey string  `json:"requirement_key"`
	NodeID         string  `json:"node_id"`
	NodeTitle      string  `json:"node_title"`
	LinkedRevision int64   `json:"linked_revision"`
	LinkedBy       string  `json:"linked_by"`
	LinkedAt       string  `json:"linked_at"`
	UnlinkedAt     *string `json:"unlinked_at"`
}

type ProjectRequirementAccessProjection struct {
	CanEdit          bool `json:"can_edit"`
	CanApprove       bool `json:"can_approve"`
	CanManageAccess  bool `json:"can_manage_access"`
	CanManageOutline bool `json:"can_manage_outline"`
}

type ProjectRequirementGrant struct {
	MemberID  string `json:"member_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	GrantKind string `json:"grant_kind"`
	GrantedBy string `json:"granted_by"`
	GrantedAt string `json:"granted_at"`
}

type ProjectRequirementAccessSet struct {
	Revision int64                     `json:"revision"`
	Grants   []ProjectRequirementGrant `json:"grants"`
}

type ProjectOutlineNode struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
}

type ProjectOutline struct {
	Revision int64                `json:"revision"`
	Nodes    []ProjectOutlineNode `json:"nodes"`
}

type ProjectRequirementBaselineResponse struct {
	Baseline         *ProjectRequirementBaseline        `json:"baseline"`
	CurrentContent   *ProjectRequirementContent         `json:"current_content"`
	EffectiveContent *ProjectRequirementContent         `json:"effective_content"`
	History          []ProjectRequirementRevision       `json:"history"`
	IssueLinks       []ProjectRequirementIssueLink      `json:"issue_links"`
	OutlineLinks     []ProjectRequirementOutlineLink    `json:"outline_links"`
	Access           ProjectRequirementAccessProjection `json:"access"`
}

type ProjectRequirementCoverageIssue struct {
	ID               string  `json:"id"`
	Identifier       string  `json:"identifier"`
	Title            string  `json:"title"`
	Status           string  `json:"status"`
	AcceptanceResult *string `json:"acceptance_result"`
}

type ProjectRequirementCoverageItem struct {
	RequirementKey string                            `json:"requirement_key"`
	Section        string                            `json:"section"`
	Text           string                            `json:"text"`
	Stage          string                            `json:"stage"`
	Issues         []ProjectRequirementCoverageIssue `json:"issues"`
}

type ProjectRequirementCoverageSnapshot struct {
	Revision    int64                            `json:"revision"`
	State       string                           `json:"state"`
	Total       int                              `json:"total"`
	Linked      int                              `json:"linked"`
	Implemented int                              `json:"implemented"`
	Accepted    int                              `json:"accepted"`
	Unlinked    int                              `json:"unlinked"`
	Items       []ProjectRequirementCoverageItem `json:"items"`
}

type ProjectRequirementCoverage struct {
	BaselineStatus *string                             `json:"baseline_status"`
	Current        *ProjectRequirementCoverageSnapshot `json:"current"`
	Effective      *ProjectRequirementCoverageSnapshot `json:"effective"`
}

type SaveProjectRequirementDraftRequest struct {
	ExpectedRevision int64                     `json:"expected_revision"`
	Content          ProjectRequirementContent `json:"content"`
	ChangeSummary    string                    `json:"change_summary"`
	MaterialChange   bool                      `json:"material_change,omitempty"`
}

type ProjectRequirementTransitionRequest struct {
	ExpectedRevision int64 `json:"expected_revision"`
}

type ProjectRequirementIssueLinkRequest struct {
	ExpectedRevision int64  `json:"expected_revision"`
	RequirementKey   string `json:"requirement_key"`
	IssueID          string `json:"issue_id"`
}

type ProjectRequirementOutlineLinkRequest struct {
	ExpectedRevision int64  `json:"expected_revision"`
	RequirementKey   string `json:"requirement_key"`
	NodeID           string `json:"node_id"`
}

type ProjectRequirementGrantInput struct {
	MemberID  string `json:"member_id"`
	GrantKind string `json:"grant_kind"`
}

type ReplaceProjectRequirementAccessRequest struct {
	ExpectedRevision int64                          `json:"expected_revision"`
	Grants           []ProjectRequirementGrantInput `json:"grants"`
}

type CreateProjectOutlineNodeRequest struct {
	ExpectedRevision int64  `json:"expected_revision"`
	Title            string `json:"title"`
}

type ProjectRequirementService interface {
	GetProjectRequirement(context.Context, string, string) (ProjectRequirementBaselineResponse, error)
	GetProjectRequirementCoverage(context.Context, string, string) (ProjectRequirementCoverage, error)
	SaveProjectRequirement(context.Context, string, string, string, SaveProjectRequirementDraftRequest) (ProjectRequirementBaselineResponse, error)
	TransitionProjectRequirement(context.Context, string, string, string, ProjectRequirementTransitionRequest) (ProjectRequirementBaselineResponse, error)
	MutateProjectRequirementIssueLink(context.Context, string, string, ProjectRequirementIssueLinkRequest, bool) (ProjectRequirementBaselineResponse, error)
	MutateProjectRequirementOutlineLink(context.Context, string, string, ProjectRequirementOutlineLinkRequest, bool) (ProjectRequirementBaselineResponse, error)
	GetProjectRequirementAccess(context.Context, string, string) (ProjectRequirementAccessSet, error)
	ReplaceProjectRequirementAccess(context.Context, string, string, ReplaceProjectRequirementAccessRequest) (ProjectRequirementAccessSet, error)
	GetProjectOutline(context.Context, string, string) (ProjectOutline, error)
	CreateProjectOutlineNode(context.Context, string, string, string, CreateProjectOutlineNodeRequest) (ProjectOutline, error)
}
