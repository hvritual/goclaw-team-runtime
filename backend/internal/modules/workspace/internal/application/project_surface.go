package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

var ErrInvalidProjectSurfaceRequest = errors.New("invalid project surface request")
var ErrProjectSurfaceNotFound = errors.New("project not found")
var ErrPinConflict = errors.New("item already pinned")
var ErrPinTargetNotFound = errors.New("pin target not found")

type ProjectSurfaceRepository interface {
	ListProjects(context.Context, string, string) ([]contract.ProjectSurfaceProject, error)
	GetProject(context.Context, string, string) (contract.ProjectSurfaceProject, error)
	CreateProject(context.Context, contract.ProjectSurfaceProject) (contract.ProjectSurfaceProject, error)
	UpdateProject(context.Context, contract.ProjectSurfaceProject) (contract.ProjectSurfaceProject, error)
	DeleteProject(context.Context, string, string, time.Time) error
	ListPins(context.Context, string, string) ([]contract.Pin, error)
	InspectPin(context.Context, string, string, string, string) (bool, bool, error)
	CreatePin(context.Context, contract.Pin) (contract.Pin, error)
	DeletePin(context.Context, string, string, string, string) error
}

type ProjectSurfaceUseCase struct {
	repository  ProjectSurfaceRepository
	authorizer  contract.WorkspaceAccessAuthorizer
	actors      contract.WorkspaceActorReader
	memberships contract.WorkspaceMembershipReader
	newID       func(context.Context) (string, error)
	now         func() time.Time
}

func NewProjectSurfaceUseCase(repository ProjectSurfaceRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, memberships contract.WorkspaceMembershipReader, newID func(context.Context) (string, error), now func() time.Time) (*ProjectSurfaceUseCase, error) {
	if repository == nil || authorizer == nil || actors == nil || memberships == nil || newID == nil || now == nil {
		return nil, errors.New("project surface dependencies are required")
	}
	return &ProjectSurfaceUseCase{repository: repository, authorizer: authorizer, actors: actors, memberships: memberships, newID: newID, now: now}, nil
}

func (u *ProjectSurfaceUseCase) ListProjects(ctx context.Context, workspaceID, status string) (contract.ProjectSurfaceList, error) {
	workspaceID, status = strings.TrimSpace(workspaceID), strings.TrimSpace(status)
	if workspaceID == "" {
		return contract.ProjectSurfaceList{}, ErrInvalidProjectSurfaceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectList); err != nil {
		return contract.ProjectSurfaceList{}, err
	}
	if status != "" && !validProjectStatus(status) {
		return contract.ProjectSurfaceList{}, ErrInvalidProjectSurfaceRequest
	}
	projects, err := u.repository.ListProjects(ctx, workspaceID, status)
	if err != nil {
		return contract.ProjectSurfaceList{}, err
	}
	if projects == nil {
		projects = []contract.ProjectSurfaceProject{}
	}
	return contract.ProjectSurfaceList{Projects: projects, Total: len(projects)}, nil
}

func (u *ProjectSurfaceUseCase) GetProject(ctx context.Context, workspaceID, projectID string) (contract.ProjectSurfaceProject, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	if workspaceID == "" || projectID == "" {
		return contract.ProjectSurfaceProject{}, ErrInvalidProjectSurfaceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectGet); err != nil {
		return contract.ProjectSurfaceProject{}, err
	}
	return u.repository.GetProject(ctx, workspaceID, projectID)
}

func (u *ProjectSurfaceUseCase) CreateProject(ctx context.Context, workspaceID string, request contract.CreateProjectSurfaceRequest) (contract.ProjectSurfaceProject, error) {
	workspaceID, request.Title = strings.TrimSpace(workspaceID), strings.TrimSpace(request.Title)
	request.Icon = normalizedProjectOptional(request.Icon)
	request.LeadType, request.LeadID = normalizedProjectOptional(request.LeadType), normalizedProjectOptional(request.LeadID)
	request.StartDate, request.DueDate = normalizedProjectOptional(request.StartDate), normalizedProjectOptional(request.DueDate)
	if request.Status == "" {
		request.Status = "planned"
	}
	if request.Priority == "" {
		request.Priority = "none"
	}
	if workspaceID == "" || request.Title == "" || !validProjectStatus(request.Status) || !validProjectPriority(request.Priority) || !validProjectDates(request.StartDate, request.DueDate) {
		return contract.ProjectSurfaceProject{}, ErrInvalidProjectSurfaceRequest
	}
	if err := validateLead(ctx, u.memberships, workspaceID, request.LeadType, request.LeadID); err != nil {
		return contract.ProjectSurfaceProject{}, err
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectCreate); err != nil {
		return contract.ProjectSurfaceProject{}, err
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.ProjectSurfaceProject{}, fmt.Errorf("generate project surface id: %w", err)
	}
	now := u.now().UTC().Format(time.RFC3339Nano)
	return u.repository.CreateProject(ctx, contract.ProjectSurfaceProject{
		ID: id, WorkspaceID: workspaceID, Title: request.Title, Description: request.Description, Icon: request.Icon,
		Status: request.Status, Priority: request.Priority, LeadType: request.LeadType, LeadID: request.LeadID,
		StartDate: request.StartDate, DueDate: request.DueDate, CreatedAt: now, UpdatedAt: now,
	})
}

func (u *ProjectSurfaceUseCase) UpdateProject(ctx context.Context, workspaceID, projectID string, request contract.UpdateProjectSurfaceRequest) (contract.ProjectSurfaceProject, error) {
	current, err := u.GetProject(ctx, workspaceID, projectID)
	if err != nil {
		return contract.ProjectSurfaceProject{}, err
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectUpdate); err != nil {
		return contract.ProjectSurfaceProject{}, err
	}
	if request.Title != nil {
		current.Title = strings.TrimSpace(*request.Title)
	}
	if request.Description.Set {
		current.Description = request.Description.Value
	}
	if request.Icon.Set {
		current.Icon = normalizedProjectOptional(request.Icon.Value)
	}
	if request.Status != nil {
		current.Status = strings.TrimSpace(*request.Status)
	}
	if request.Priority != nil {
		current.Priority = strings.TrimSpace(*request.Priority)
	}
	if request.LeadType.Set {
		current.LeadType = normalizedProjectOptional(request.LeadType.Value)
	}
	if request.LeadID.Set {
		current.LeadID = normalizedProjectOptional(request.LeadID.Value)
	}
	if request.StartDate.Set {
		current.StartDate = normalizedProjectOptional(request.StartDate.Value)
	}
	if request.DueDate.Set {
		current.DueDate = normalizedProjectOptional(request.DueDate.Value)
	}
	if current.Title == "" || !validProjectStatus(current.Status) || !validProjectPriority(current.Priority) || !validProjectDates(current.StartDate, current.DueDate) {
		return contract.ProjectSurfaceProject{}, ErrInvalidProjectSurfaceRequest
	}
	if err := validateLead(ctx, u.memberships, workspaceID, current.LeadType, current.LeadID); err != nil {
		return contract.ProjectSurfaceProject{}, err
	}
	current.UpdatedAt = u.now().UTC().Format(time.RFC3339Nano)
	return u.repository.UpdateProject(ctx, current)
}

func (u *ProjectSurfaceUseCase) DeleteProject(ctx context.Context, workspaceID, projectID string) error {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	if workspaceID == "" || projectID == "" {
		return ErrInvalidProjectSurfaceRequest
	}
	if _, err := u.GetProject(ctx, workspaceID, projectID); err != nil {
		return err
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionProjectDelete); err != nil {
		return err
	}
	return u.repository.DeleteProject(ctx, workspaceID, projectID, u.now())
}

func (u *ProjectSurfaceUseCase) ListPins(ctx context.Context, workspaceID, userID string) ([]contract.Pin, error) {
	workspaceID, userID = strings.TrimSpace(workspaceID), strings.TrimSpace(userID)
	if workspaceID == "" || userID == "" {
		return nil, ErrInvalidProjectSurfaceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.pin.list"); err != nil {
		return nil, err
	}
	pins, err := u.repository.ListPins(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if pins == nil {
		pins = []contract.Pin{}
	}
	return pins, nil
}

func (u *ProjectSurfaceUseCase) CreatePin(ctx context.Context, workspaceID, userID string, request contract.CreatePinRequest) (contract.Pin, error) {
	workspaceID, userID = strings.TrimSpace(workspaceID), strings.TrimSpace(userID)
	request.ItemType, request.ItemID = strings.TrimSpace(request.ItemType), strings.TrimSpace(request.ItemID)
	if workspaceID == "" || userID == "" || (request.ItemType != "issue" && request.ItemType != "project") || request.ItemID == "" {
		return contract.Pin{}, ErrInvalidProjectSurfaceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.pin.create"); err != nil {
		return contract.Pin{}, err
	}
	targetExists, alreadyPinned, err := u.repository.InspectPin(ctx, workspaceID, userID, request.ItemType, request.ItemID)
	if err != nil {
		return contract.Pin{}, err
	}
	if !targetExists {
		return contract.Pin{}, ErrPinTargetNotFound
	}
	if alreadyPinned {
		return contract.Pin{}, ErrPinConflict
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.Pin{}, fmt.Errorf("generate pin id: %w", err)
	}
	return u.repository.CreatePin(ctx, contract.Pin{ID: id, WorkspaceID: workspaceID, UserID: userID, ItemType: request.ItemType, ItemID: request.ItemID, CreatedAt: u.now().UTC().Format(time.RFC3339Nano)})
}

func (u *ProjectSurfaceUseCase) DeletePin(ctx context.Context, workspaceID, userID, itemType, itemID string) error {
	workspaceID, userID, itemType, itemID = strings.TrimSpace(workspaceID), strings.TrimSpace(userID), strings.TrimSpace(itemType), strings.TrimSpace(itemID)
	if workspaceID == "" || userID == "" || (itemType != "issue" && itemType != "project") || itemID == "" {
		return ErrInvalidProjectSurfaceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, "workspace.pin.delete"); err != nil {
		return err
	}
	return u.repository.DeletePin(ctx, workspaceID, userID, itemType, itemID)
}

func validProjectStatus(value string) bool {
	switch value {
	case "planned", "in_progress", "paused", "completed", "cancelled":
		return true
	}
	return false
}

func validProjectPriority(value string) bool {
	switch value {
	case "urgent", "high", "medium", "low", "none":
		return true
	}
	return false
}

func validProjectDates(values ...*string) bool {
	for _, value := range values {
		if value != nil && *value != "" {
			if _, err := time.Parse("2006-01-02", *value); err != nil {
				return false
			}
		}
	}
	return true
}

func normalizedProjectOptional(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func validateLead(ctx context.Context, memberships contract.WorkspaceMembershipReader, workspaceID string, leadType, leadID *string) error {
	if leadType == nil && leadID == nil {
		return nil
	}
	if leadType == nil || leadID == nil || *leadType != "member" || strings.TrimSpace(*leadID) == "" {
		return ErrInvalidProjectSurfaceRequest
	}
	_, ok, err := memberships.FindForUserAndWorkspace(ctx, strings.TrimSpace(*leadID), workspaceID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidProjectSurfaceRequest
	}
	return nil
}

var _ contract.ProjectSurfaceService = (*ProjectSurfaceUseCase)(nil)
