package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

const (
	PermissionIssueCreate       = "workspace.issue.create"
	PermissionIssueGet          = "workspace.issue.get"
	PermissionIssueList         = "workspace.issue.list"
	PermissionIssueUpdate       = "workspace.issue.update"
	PermissionIssueUpdateStatus = "workspace.issue.update_status"
)

var ErrIssueRecordNotFound = errors.New("issue record not found")

type IssueListQuery struct {
	WorkspaceID              string
	ProjectID, ParentIssueID *string
	Status, Priority         string
	AssigneeType, AssigneeID *string
	CreatorType, CreatorID   *string
}

type IssueReferenceRepository interface {
	FindByIDOrIdentifier(context.Context, string, string) (issueDomain.Issue, error)
}

type IssueRepository interface {
	IssueReferenceRepository
	Create(context.Context, issueDomain.Issue) (issueDomain.Issue, error)
	List(context.Context, IssueListQuery) ([]issueDomain.Issue, error)
	Update(context.Context, issueDomain.Issue) error
	WouldCreateParentCycle(context.Context, string, string, string) (bool, error)
}

type IssueUseCase struct {
	repository IssueRepository
	projects   ProjectRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
	assets     contract.WorkspaceAssetReader
	newID      ProjectIDGenerator
	now        Clock
}

func NewIssueUseCase(repository IssueRepository, projects ProjectRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, assets contract.WorkspaceAssetReader, newID ProjectIDGenerator, now Clock) (*IssueUseCase, error) {
	if repository == nil || projects == nil || authorizer == nil || actors == nil || assets == nil || newID == nil || now == nil {
		return nil, errors.New("Issue dependencies are required")
	}
	return &IssueUseCase{repository: repository, projects: projects, authorizer: authorizer, actors: actors, assets: assets, newID: newID, now: now}, nil
}

func (s *IssueUseCase) CreateIssue(ctx context.Context, request contract.CreateIssueRequest) (contract.CreateIssueResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.CreateIssueResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueCreate); err != nil {
		return contract.CreateIssueResponse{}, err
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return contract.CreateIssueResponse{}, contract.ErrWorkspaceActorRequired
	}
	if err := s.requireActor(ctx, workspaceID, actor.Type, actor.ID); err != nil {
		return contract.CreateIssueResponse{}, err
	}
	projectID := cleanOptionalString(request.ProjectId)
	if err := s.validateProject(ctx, workspaceID, projectID); err != nil {
		return contract.CreateIssueResponse{}, err
	}
	parentID, err := s.canonicalParent(ctx, workspaceID, cleanOptionalString(request.ParentIssueId))
	if err != nil {
		return contract.CreateIssueResponse{}, err
	}
	if err := s.validateActorPair(ctx, workspaceID, request.AssigneeType, request.AssigneeId); err != nil {
		return contract.CreateIssueResponse{}, err
	}
	if err := s.validateAssets(ctx, workspaceID, request.AssetIds); err != nil {
		return contract.CreateIssueResponse{}, err
	}
	id, err := s.newID(ctx)
	if err != nil {
		return contract.CreateIssueResponse{}, fmt.Errorf("generate Issue id: %w", err)
	}
	value, err := issueDomain.New(
		id, workspaceID, request.Title, request.Description, request.Status, request.Priority,
		request.AssigneeType, request.AssigneeId, parentID, projectID, actor.Type, actor.ID,
		request.Position, request.Stage, request.StartDate, request.DueDate, request.AssetIds, s.now(),
	)
	if err != nil {
		return contract.CreateIssueResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidIssue, err)
	}
	created, err := s.repository.Create(ctx, value)
	if err != nil {
		return contract.CreateIssueResponse{}, fmt.Errorf("create Issue: %w", err)
	}
	result := issueToContract(created)
	return contract.CreateIssueResponse{Issue: &result}, nil
}

func (s *IssueUseCase) GetIssue(ctx context.Context, request contract.GetIssueRequest) (contract.GetIssueResponse, error) {
	workspaceID, issueID, err := validateIssueIdentity(request.WorkspaceId, request.IssueId)
	if err != nil {
		return contract.GetIssueResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueGet); err != nil {
		return contract.GetIssueResponse{}, err
	}
	value, err := s.findIssue(ctx, workspaceID, issueID)
	if err != nil {
		return contract.GetIssueResponse{}, err
	}
	result := issueToContract(value)
	return contract.GetIssueResponse{Issue: &result}, nil
}

func (s *IssueUseCase) ListIssues(ctx context.Context, request contract.ListIssuesRequest) (contract.ListIssuesResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.ListIssuesResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidIssue)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueList); err != nil {
		return contract.ListIssuesResponse{}, err
	}
	if request.Status != "" && !issueDomain.ValidStatus(request.Status) {
		return contract.ListIssuesResponse{}, fmt.Errorf("%w: invalid status", contract.ErrInvalidIssue)
	}
	if request.Priority != "" && !issueDomain.ValidPriority(request.Priority) {
		return contract.ListIssuesResponse{}, fmt.Errorf("%w: invalid priority", contract.ErrInvalidIssue)
	}
	if err := validateOptionalActorFilter(request.AssigneeType, request.AssigneeId); err != nil {
		return contract.ListIssuesResponse{}, err
	}
	if err := validateOptionalActorFilter(request.CreatorType, request.CreatorId); err != nil {
		return contract.ListIssuesResponse{}, err
	}
	values, err := s.repository.List(ctx, IssueListQuery{
		WorkspaceID: workspaceID, ProjectID: cleanOptionalString(request.ProjectId), ParentIssueID: cleanOptionalString(request.ParentIssueId),
		Status: request.Status, Priority: request.Priority,
		AssigneeType: cleanOptionalString(request.AssigneeType), AssigneeID: cleanOptionalString(request.AssigneeId),
		CreatorType: cleanOptionalString(request.CreatorType), CreatorID: cleanOptionalString(request.CreatorId),
	})
	if err != nil {
		return contract.ListIssuesResponse{}, fmt.Errorf("list Issues: %w", err)
	}
	result := make([]contract.Issue, len(values))
	for index := range values {
		result[index] = issueToContract(values[index])
	}
	return contract.ListIssuesResponse{Issues: result, Total: countToInt32(len(result))}, nil
}

func (s *IssueUseCase) UpdateIssue(ctx context.Context, request contract.UpdateIssueRequest) (contract.UpdateIssueResponse, error) {
	return s.updateIssue(ctx, request, PermissionIssueUpdate)
}

func (s *IssueUseCase) UpdateIssueStatus(ctx context.Context, request contract.UpdateIssueStatusRequest) (contract.UpdateIssueStatusResponse, error) {
	status := request.Status
	updated, err := s.updateIssue(ctx, contract.UpdateIssueRequest{WorkspaceId: request.WorkspaceId, IssueId: request.IssueId, Status: &status}, PermissionIssueUpdateStatus)
	if err != nil {
		return contract.UpdateIssueStatusResponse{}, err
	}
	return contract.UpdateIssueStatusResponse{Issue: updated.Issue}, nil
}

func (s *IssueUseCase) updateIssue(ctx context.Context, request contract.UpdateIssueRequest, permission string) (contract.UpdateIssueResponse, error) {
	workspaceID, issueID, err := validateIssueIdentity(request.WorkspaceId, request.IssueId)
	if err != nil {
		return contract.UpdateIssueResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission); err != nil {
		return contract.UpdateIssueResponse{}, err
	}
	value, err := s.findIssue(ctx, workspaceID, issueID)
	if err != nil {
		return contract.UpdateIssueResponse{}, err
	}
	patch := issueDomain.Patch{
		Title: request.Title, Description: request.Description, Status: request.Status, Priority: request.Priority,
		AssigneeType: issueStringChange(request.AssigneeType), AssigneeID: issueStringChange(request.AssigneeId),
		ParentIssueID: issueStringChange(request.ParentIssueId), ProjectID: issueStringChange(request.ProjectId),
		Position: request.Position, StartDate: issueDateChange(request.StartDate), DueDate: issueDateChange(request.DueDate),
	}
	if request.Stage != nil {
		if *request.Stage < 0 {
			return contract.UpdateIssueResponse{}, fmt.Errorf("%w: stage cannot be negative", contract.ErrInvalidIssue)
		}
		patch.Stage.Set = true
		if *request.Stage > 0 {
			stage := *request.Stage
			patch.Stage.Value = &stage
		}
	}
	if request.AssetIds != nil {
		patch.AssetIDs = issueDomain.AssetsChange{Set: true, Values: append([]string(nil), request.AssetIds.Values...)}
		if err := s.validateAssets(ctx, workspaceID, patch.AssetIDs.Values); err != nil {
			return contract.UpdateIssueResponse{}, err
		}
	}
	if request.ProjectId != nil {
		if err := s.validateProject(ctx, workspaceID, patch.ProjectID.Value); err != nil {
			return contract.UpdateIssueResponse{}, err
		}
	}
	if request.ParentIssueId != nil && patch.ParentIssueID.Value != nil {
		parentID, canonicalErr := s.canonicalParent(ctx, workspaceID, patch.ParentIssueID.Value)
		if canonicalErr != nil {
			return contract.UpdateIssueResponse{}, canonicalErr
		}
		patch.ParentIssueID.Value = parentID
		cycle, cycleErr := s.repository.WouldCreateParentCycle(ctx, workspaceID, value.ID, *parentID)
		if cycleErr != nil {
			return contract.UpdateIssueResponse{}, fmt.Errorf("validate Issue parent cycle: %w", cycleErr)
		}
		if cycle {
			return contract.UpdateIssueResponse{}, fmt.Errorf("%w: circular parent relationship", contract.ErrInvalidIssue)
		}
	}
	if request.AssigneeType != nil || request.AssigneeId != nil {
		candidateType, candidateID := value.AssigneeType, value.AssigneeID
		if patch.AssigneeType.Set {
			candidateType = patch.AssigneeType.Value
		}
		if patch.AssigneeID.Set {
			candidateID = patch.AssigneeID.Value
		}
		if err := s.validateActorPair(ctx, workspaceID, candidateType, candidateID); err != nil {
			return contract.UpdateIssueResponse{}, err
		}
	}
	updated, err := value.Apply(patch, s.now())
	if err != nil {
		return contract.UpdateIssueResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidIssue, err)
	}
	if err := s.repository.Update(ctx, updated); errors.Is(err, ErrIssueRecordNotFound) {
		return contract.UpdateIssueResponse{}, contract.ErrIssueNotFound
	} else if err != nil {
		return contract.UpdateIssueResponse{}, fmt.Errorf("update Issue: %w", err)
	}
	result := issueToContract(updated)
	return contract.UpdateIssueResponse{Issue: &result}, nil
}

func (s *IssueUseCase) findIssue(ctx context.Context, workspaceID, issueID string) (issueDomain.Issue, error) {
	value, err := s.repository.FindByIDOrIdentifier(ctx, workspaceID, issueID)
	if errors.Is(err, ErrIssueRecordNotFound) {
		return issueDomain.Issue{}, contract.ErrIssueNotFound
	}
	if err != nil {
		return issueDomain.Issue{}, fmt.Errorf("get Issue: %w", err)
	}
	return value, nil
}

func (s *IssueUseCase) validateProject(ctx context.Context, workspaceID string, projectID *string) error {
	if projectID == nil {
		return nil
	}
	if _, err := s.projects.FindByID(ctx, workspaceID, *projectID); errors.Is(err, ErrProjectRecordNotFound) {
		return contract.ErrProjectNotFound
	} else if err != nil {
		return fmt.Errorf("validate Issue Project: %w", err)
	}
	return nil
}

func (s *IssueUseCase) canonicalParent(ctx context.Context, workspaceID string, parentID *string) (*string, error) {
	if parentID == nil {
		return nil, nil
	}
	parent, err := s.repository.FindByIDOrIdentifier(ctx, workspaceID, *parentID)
	if errors.Is(err, ErrIssueRecordNotFound) {
		return nil, contract.ErrIssueNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate parent Issue: %w", err)
	}
	canonical := parent.ID
	return &canonical, nil
}

func (s *IssueUseCase) validateActorPair(ctx context.Context, workspaceID string, actorType, actorID *string) error {
	actorType, actorID = cleanOptionalString(actorType), cleanOptionalString(actorID)
	if (actorType == nil) != (actorID == nil) {
		return fmt.Errorf("%w: assignee type and id must be paired", contract.ErrInvalidIssue)
	}
	if actorType == nil {
		return nil
	}
	return s.requireActor(ctx, workspaceID, *actorType, *actorID)
}

func (s *IssueUseCase) requireActor(ctx context.Context, workspaceID, actorType, actorID string) error {
	if actorType != "member" && actorType != "agent" {
		return fmt.Errorf("%w: actor type must be member or agent", contract.ErrInvalidIssue)
	}
	belongs, err := s.actors.ActorBelongsToWorkspace(ctx, workspaceID, actorType, actorID)
	if err != nil {
		return fmt.Errorf("validate Issue actor: %w", err)
	}
	if !belongs {
		return contract.ErrActorOutsideWorkspace
	}
	return nil
}

func (s *IssueUseCase) validateAssets(ctx context.Context, workspaceID string, assetIDs []string) error {
	seen := make(map[string]struct{}, len(assetIDs))
	for _, assetID := range assetIDs {
		if assetID == "" || assetID != strings.TrimSpace(assetID) {
			return fmt.Errorf("%w: asset id is required", contract.ErrInvalidIssue)
		}
		if _, duplicate := seen[assetID]; duplicate {
			return fmt.Errorf("%w: duplicate asset id", contract.ErrInvalidIssue)
		}
		seen[assetID] = struct{}{}
		belongs, err := s.assets.AssetBelongsToWorkspace(ctx, workspaceID, assetID)
		if err != nil {
			return fmt.Errorf("validate Issue Asset: %w", err)
		}
		if !belongs {
			return contract.ErrAssetOutsideWorkspace
		}
	}
	return nil
}

func validateIssueIdentity(workspaceID, issueID string) (string, string, error) {
	workspaceID, issueID = strings.TrimSpace(workspaceID), strings.TrimSpace(issueID)
	if workspaceID == "" || issueID == "" {
		return "", "", fmt.Errorf("%w: workspace id and Issue id are required", contract.ErrInvalidIssue)
	}
	return workspaceID, issueID, nil
}

func validateOptionalActorFilter(actorType, actorID *string) error {
	actorType, actorID = cleanOptionalString(actorType), cleanOptionalString(actorID)
	if (actorType == nil) != (actorID == nil) {
		return fmt.Errorf("%w: actor filter type and id must be paired", contract.ErrInvalidIssue)
	}
	if actorType != nil && *actorType != "member" && *actorType != "agent" {
		return fmt.Errorf("%w: actor filter type must be member or agent", contract.ErrInvalidIssue)
	}
	return nil
}

func issueStringChange(value *string) issueDomain.StringChange {
	if value == nil {
		return issueDomain.StringChange{}
	}
	return issueDomain.StringChange{Set: true, Value: cleanOptionalString(value)}
}

func issueDateChange(value *string) issueDomain.StringChange {
	if value == nil {
		return issueDomain.StringChange{}
	}
	if *value == "" {
		return issueDomain.StringChange{Set: true}
	}
	copied := *value
	return issueDomain.StringChange{Set: true, Value: &copied}
}

func issueToContract(value issueDomain.Issue) contract.Issue {
	return contract.Issue{
		Id: value.ID, WorkspaceId: value.WorkspaceID, Number: value.Number, Identifier: value.Identifier,
		Title: value.Title, Description: copyTodoString(value.Description), Status: value.Status, Priority: value.Priority,
		AssigneeType: copyTodoString(value.AssigneeType), AssigneeId: copyTodoString(value.AssigneeID),
		CreatorType: value.CreatorType, CreatorId: value.CreatorID,
		ParentIssueId: copyTodoString(value.ParentIssueID), ProjectId: copyTodoString(value.ProjectID),
		Position: value.Position, Stage: copyIssueStage(value.Stage), StartDate: copyTodoString(value.StartDate), DueDate: copyTodoString(value.DueDate),
		CreatedAt: value.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.Format(time.RFC3339Nano),
		Metadata: cloneIssueMap(value.Metadata), Properties: cloneIssueMap(value.Properties), AssetIds: append([]string{}, value.AssetIDs...),
	}
}

func copyIssueStage(value *int32) *int32 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneIssueMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

var _ contract.IssueService = (*IssueUseCase)(nil)
