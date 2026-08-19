package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
)

var (
	ErrInvalidProjectRequirementRequest  = errors.New("invalid Project Requirement request")
	ErrProjectRequirementNotFound        = errors.New("Project Requirement not found")
	ErrProjectRequirementConflict        = errors.New("Project Requirement conflict")
	ErrProjectRequirementTransition      = errors.New("invalid Project Requirement transition")
	ErrProjectRequirementSelfApproval    = errors.New("independent Project Requirement approval required")
	ErrLegacyRequirementMutationDisabled = contract.ErrLegacyRequirementMutationDisabled
)

type ProjectRequirementSave struct {
	BaselineID       string
	WorkspaceID      string
	ProjectID        string
	ExpectedRevision int64
	Content          requirementDomain.Content
	ChangeSummary    string
	MaterialChange   bool
	IdempotencyKey   string
	RequestHash      string
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectRequirementTransition struct {
	WorkspaceID      string
	ProjectID        string
	Action           string
	ExpectedRevision int64
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectRequirementLinkMutation struct {
	WorkspaceID      string
	ProjectID        string
	RequirementKey   string
	TargetKind       string
	TargetID         string
	ExpectedRevision int64
	Unlink           bool
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectRequirementGrantChange struct {
	MemberID  string
	GrantKind string
}

type ProjectRequirementAccessReplace struct {
	WorkspaceID      string
	ProjectID        string
	ExpectedRevision int64
	Grants           []ProjectRequirementGrantChange
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectOutlineNodeCreate struct {
	NodeID           string
	WorkspaceID      string
	ProjectID        string
	ExpectedRevision int64
	Title            string
	IdempotencyKey   string
	RequestHash      string
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectRequirementRepository interface {
	ReadProjectRequirement(context.Context, string, string, contract.WorkspaceActor) (contract.ProjectRequirementBaselineResponse, error)
	SaveProjectRequirement(context.Context, ProjectRequirementSave) (contract.ProjectRequirementBaselineResponse, error)
	TransitionProjectRequirement(context.Context, ProjectRequirementTransition) (contract.ProjectRequirementBaselineResponse, error)
	MutateProjectRequirementLink(context.Context, ProjectRequirementLinkMutation) (contract.ProjectRequirementBaselineResponse, error)
	ReplaceProjectRequirementAccess(context.Context, ProjectRequirementAccessReplace) (contract.ProjectRequirementAccessSet, error)
	ReadProjectRequirementAccess(context.Context, string, string, contract.WorkspaceActor) (contract.ProjectRequirementAccessSet, error)
	CreateProjectOutlineNode(context.Context, ProjectOutlineNodeCreate) (contract.ProjectOutline, error)
	ReadProjectOutline(context.Context, string, string, contract.WorkspaceActor) (contract.ProjectOutline, error)
}

type ProjectRequirementUseCase struct {
	repository       ProjectRequirementRepository
	newBaselineID    func(context.Context) (string, error)
	newOutlineNodeID func(context.Context) (string, error)
	now              func() time.Time
}

func NewProjectRequirementUseCase(repository ProjectRequirementRepository, newBaselineID, newOutlineNodeID func(context.Context) (string, error), now func() time.Time) (*ProjectRequirementUseCase, error) {
	if repository == nil || newBaselineID == nil || newOutlineNodeID == nil || now == nil {
		return nil, errors.New("Project Requirement dependencies are required")
	}
	return &ProjectRequirementUseCase{repository: repository, newBaselineID: newBaselineID, newOutlineNodeID: newOutlineNodeID, now: now}, nil
}

func (u *ProjectRequirementUseCase) GetProjectRequirement(ctx context.Context, workspaceID, projectID string) (contract.ProjectRequirementBaselineResponse, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if workspaceID == "" || projectID == "" {
		return contract.ProjectRequirementBaselineResponse{}, ErrInvalidProjectRequirementRequest
	}
	return u.repository.ReadProjectRequirement(ctx, workspaceID, projectID, actor)
}

func (u *ProjectRequirementUseCase) SaveProjectRequirement(ctx context.Context, workspaceID, projectID, idempotencyKey string, request contract.SaveProjectRequirementDraftRequest) (contract.ProjectRequirementBaselineResponse, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if workspaceID == "" || projectID == "" || request.ExpectedRevision < 0 {
		return contract.ProjectRequirementBaselineResponse{}, ErrInvalidProjectRequirementRequest
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if request.ExpectedRevision == 0 && (idempotencyKey == "" || len(idempotencyKey) > 200) {
		return contract.ProjectRequirementBaselineResponse{}, ErrInvalidProjectRequirementRequest
	}
	content := projectRequirementContentToDomain(request.Content)
	changeSummary := strings.TrimSpace(request.ChangeSummary)
	requestHash, err := projectRequirementRequestHash(struct {
		WorkspaceID      string
		ProjectID        string
		ExpectedRevision int64
		Content          requirementDomain.Content
		ChangeSummary    string
		MaterialChange   bool
	}{workspaceID, projectID, request.ExpectedRevision, content, changeSummary, request.MaterialChange})
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	baselineID := ""
	if request.ExpectedRevision == 0 {
		baselineID, err = u.newBaselineID(ctx)
		if err != nil {
			return contract.ProjectRequirementBaselineResponse{}, err
		}
	}
	return u.repository.SaveProjectRequirement(ctx, ProjectRequirementSave{
		BaselineID: baselineID, WorkspaceID: workspaceID, ProjectID: projectID,
		ExpectedRevision: request.ExpectedRevision, Content: content, ChangeSummary: changeSummary,
		MaterialChange: request.MaterialChange, IdempotencyKey: idempotencyKey, RequestHash: requestHash,
		Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectRequirementUseCase) TransitionProjectRequirement(ctx context.Context, workspaceID, projectID, action string, request contract.ProjectRequirementTransitionRequest) (contract.ProjectRequirementBaselineResponse, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	actions := map[string]string{
		"submit-review": "submit_review",
		"withdraw":      "withdraw_review",
		"approve":       "approve",
		"freeze":        "freeze",
		"retire":        "retire",
	}
	internalAction, ok := actions[strings.TrimSpace(action)]
	if workspaceID == "" || projectID == "" || request.ExpectedRevision < 1 || !ok {
		return contract.ProjectRequirementBaselineResponse{}, ErrInvalidProjectRequirementRequest
	}
	return u.repository.TransitionProjectRequirement(ctx, ProjectRequirementTransition{
		WorkspaceID: workspaceID, ProjectID: projectID, Action: internalAction,
		ExpectedRevision: request.ExpectedRevision, Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectRequirementUseCase) MutateProjectRequirementIssueLink(ctx context.Context, workspaceID, projectID string, request contract.ProjectRequirementIssueLinkRequest, unlink bool) (contract.ProjectRequirementBaselineResponse, error) {
	return u.mutateProjectRequirementLink(ctx, workspaceID, projectID, request.RequirementKey, "issue", request.IssueID, request.ExpectedRevision, unlink)
}

func (u *ProjectRequirementUseCase) MutateProjectRequirementOutlineLink(ctx context.Context, workspaceID, projectID string, request contract.ProjectRequirementOutlineLinkRequest, unlink bool) (contract.ProjectRequirementBaselineResponse, error) {
	return u.mutateProjectRequirementLink(ctx, workspaceID, projectID, request.RequirementKey, "outline", request.NodeID, request.ExpectedRevision, unlink)
}

func (u *ProjectRequirementUseCase) mutateProjectRequirementLink(ctx context.Context, workspaceID, projectID, key, kind, targetID string, expected int64, unlink bool) (contract.ProjectRequirementBaselineResponse, error) {
	workspaceID, projectID, key, targetID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), strings.TrimSpace(key), strings.TrimSpace(targetID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if workspaceID == "" || projectID == "" || key == "" || targetID == "" || expected < 1 {
		return contract.ProjectRequirementBaselineResponse{}, ErrInvalidProjectRequirementRequest
	}
	return u.repository.MutateProjectRequirementLink(ctx, ProjectRequirementLinkMutation{
		WorkspaceID: workspaceID, ProjectID: projectID, RequirementKey: key, TargetKind: kind,
		TargetID: targetID, ExpectedRevision: expected, Unlink: unlink, Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectRequirementUseCase) GetProjectRequirementAccess(ctx context.Context, workspaceID, projectID string) (contract.ProjectRequirementAccessSet, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if workspaceID == "" || projectID == "" {
		return contract.ProjectRequirementAccessSet{}, ErrInvalidProjectRequirementRequest
	}
	return u.repository.ReadProjectRequirementAccess(ctx, workspaceID, projectID, actor)
}

func (u *ProjectRequirementUseCase) ReplaceProjectRequirementAccess(ctx context.Context, workspaceID, projectID string, request contract.ReplaceProjectRequirementAccessRequest) (contract.ProjectRequirementAccessSet, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if workspaceID == "" || projectID == "" || request.ExpectedRevision < 0 {
		return contract.ProjectRequirementAccessSet{}, ErrInvalidProjectRequirementRequest
	}
	grants := make([]ProjectRequirementGrantChange, len(request.Grants))
	for index, grant := range request.Grants {
		grants[index] = ProjectRequirementGrantChange{MemberID: strings.TrimSpace(grant.MemberID), GrantKind: strings.TrimSpace(grant.GrantKind)}
	}
	return u.repository.ReplaceProjectRequirementAccess(ctx, ProjectRequirementAccessReplace{
		WorkspaceID: workspaceID, ProjectID: projectID, ExpectedRevision: request.ExpectedRevision,
		Grants: grants, Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectRequirementUseCase) GetProjectOutline(ctx context.Context, workspaceID, projectID string) (contract.ProjectOutline, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	if workspaceID == "" || projectID == "" {
		return contract.ProjectOutline{}, ErrInvalidProjectRequirementRequest
	}
	return u.repository.ReadProjectOutline(ctx, workspaceID, projectID, actor)
}

func (u *ProjectRequirementUseCase) CreateProjectOutlineNode(ctx context.Context, workspaceID, projectID, idempotencyKey string, request contract.CreateProjectOutlineNodeRequest) (contract.ProjectOutline, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	title, idempotencyKey := strings.TrimSpace(request.Title), strings.TrimSpace(idempotencyKey)
	actor, err := projectRequirementActor(ctx)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	if workspaceID == "" || projectID == "" || request.ExpectedRevision < 0 || title == "" || len(title) > 500 || idempotencyKey == "" || len(idempotencyKey) > 200 {
		return contract.ProjectOutline{}, ErrInvalidProjectRequirementRequest
	}
	requestHash, err := projectRequirementRequestHash(struct {
		WorkspaceID      string
		ProjectID        string
		ExpectedRevision int64
		Title            string
	}{workspaceID, projectID, request.ExpectedRevision, title})
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	nodeID, err := u.newOutlineNodeID(ctx)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	return u.repository.CreateProjectOutlineNode(ctx, ProjectOutlineNodeCreate{
		NodeID: nodeID, WorkspaceID: workspaceID, ProjectID: projectID, ExpectedRevision: request.ExpectedRevision,
		Title: title, IdempotencyKey: idempotencyKey, RequestHash: requestHash, Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func projectRequirementActor(ctx context.Context) (contract.WorkspaceActor, error) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" || strings.TrimSpace(actor.ID) == "" {
		return contract.WorkspaceActor{}, contract.ErrWorkspaceActorRequired
	}
	actor.ID = strings.TrimSpace(actor.ID)
	return actor, nil
}

func projectRequirementContentToDomain(value contract.ProjectRequirementContent) requirementDomain.Content {
	convert := func(items []contract.ProjectRequirementItem) []requirementDomain.Item {
		result := make([]requirementDomain.Item, len(items))
		for index, item := range items {
			result[index] = requirementDomain.Item{Key: strings.TrimSpace(item.Key), Text: strings.TrimSpace(item.Text)}
		}
		return result
	}
	return requirementDomain.Content{
		ProblemStatement: strings.TrimSpace(value.ProblemStatement), Goals: convert(value.Goals),
		InScope: convert(value.InScope), OutOfScope: convert(value.OutOfScope), Constraints: convert(value.Constraints),
		AcceptanceCriteria: convert(value.AcceptanceCriteria), Dependencies: convert(value.Dependencies),
	}
}

func projectRequirementRequestHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

var _ contract.ProjectRequirementService = (*ProjectRequirementUseCase)(nil)
