package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/domain/projectresource"
)

var ErrInvalidProjectResourceRequest = errors.New("invalid Project Resource request")
var ErrProjectResourceNotFound = errors.New("Project Resource not found")
var ErrProjectResourceConflict = errors.New("Project Resource conflict")

var projectResourceDiagnosticPattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

type ProjectResourceProjectAccess struct {
	Status   string
	LeadType string
	LeadID   string
}

type ProjectResourceCreate struct {
	ID             string
	WorkspaceID    string
	ProjectID      string
	ResourceType   string
	ResourceRef    contract.ProjectResourceRef
	Fingerprint    string
	Label          string
	IdempotencyKey string
	RequestHash    string
	Actor          contract.WorkspaceActor
	OccurredAt     time.Time
}

type ProjectResourceMutation struct {
	WorkspaceID      string
	ProjectID        string
	ResourceID       string
	Action           string
	ExpectedRevision int64
	ResourceRef      *contract.ProjectResourceRef
	Fingerprint      string
	Label            *string
	BeforeResourceID *string
	Connection       contract.ProjectResourceConnection
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectResourceRepository interface {
	ProjectResourceAccess(context.Context, string, string) (ProjectResourceProjectAccess, error)
	ListProjectResources(context.Context, string, string, bool) (contract.ProjectResourceList, error)
	GetProjectResource(context.Context, string, string, string) (contract.ProjectResource, error)
	CreateProjectResource(context.Context, ProjectResourceCreate) (contract.ProjectResource, error)
	MutateProjectResource(context.Context, ProjectResourceMutation) (contract.ProjectResource, error)
	RefreshProjectResource(context.Context, ProjectResourceMutation, ProjectResourceConnectionResolver) (contract.ProjectResource, error)
	ArchiveProjectResource(context.Context, string, string, string, int64, contract.WorkspaceActor, time.Time) error
}

type ProjectResourceConnectionResolver func(context.Context, contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection

type ProjectResourceConnectionChecker interface {
	Check(context.Context, contract.ProjectResourceConnectionRequest) (contract.ProjectResourceConnection, error)
}

type ProjectResourceUseCase struct {
	repository  ProjectResourceRepository
	authorizer  contract.WorkspaceAccessAuthorizer
	memberships contract.WorkspaceMembershipReader
	checker     ProjectResourceConnectionChecker
	newID       func(context.Context) (string, error)
	now         func() time.Time
}

func NewProjectResourceUseCase(repository ProjectResourceRepository, authorizer contract.WorkspaceAccessAuthorizer, memberships contract.WorkspaceMembershipReader, checker ProjectResourceConnectionChecker, newID func(context.Context) (string, error), now func() time.Time) (*ProjectResourceUseCase, error) {
	if repository == nil || authorizer == nil || memberships == nil || checker == nil || newID == nil || now == nil {
		return nil, errors.New("Project Resource dependencies are required")
	}
	return &ProjectResourceUseCase{repository: repository, authorizer: authorizer, memberships: memberships, checker: checker, newID: newID, now: now}, nil
}

func (u *ProjectResourceUseCase) ListProjectResources(ctx context.Context, workspaceID, projectID string, includeArchived bool) (contract.ProjectResourceList, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	if workspaceID == "" || projectID == "" {
		return contract.ProjectResourceList{}, ErrInvalidProjectResourceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionResourceRead); err != nil {
		return contract.ProjectResourceList{}, err
	}
	if _, err := u.repository.ProjectResourceAccess(ctx, workspaceID, projectID); err != nil {
		return contract.ProjectResourceList{}, err
	}
	result, err := u.repository.ListProjectResources(ctx, workspaceID, projectID, includeArchived)
	if result.Resources == nil {
		result.Resources = []contract.ProjectResource{}
	}
	return result, err
}

func (u *ProjectResourceUseCase) CreateProjectResource(ctx context.Context, workspaceID, projectID, idempotencyKey string, request contract.CreateProjectResourceRequest) (contract.ProjectResource, error) {
	actor, _, err := u.authorizeProjectResourceManage(ctx, workspaceID, projectID)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	label := strings.TrimSpace(request.Label)
	if idempotencyKey == "" || len(idempotencyKey) > 200 || len(label) > 120 {
		return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
	}
	kind := projectresource.Type(strings.TrimSpace(request.ResourceType))
	reference, err := projectresource.Normalize(kind, request.ResourceRef.URL, request.ResourceRef.Ref)
	if err != nil {
		return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	canonical := contract.ProjectResourceRef{URL: reference.URL, Ref: reference.Ref}
	requestHash, err := projectResourceRequestHash(struct {
		WorkspaceID  string
		ProjectID    string
		ResourceType string
		ResourceRef  contract.ProjectResourceRef
		Label        string
	}{strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), string(kind), canonical, label})
	if err != nil {
		return contract.ProjectResource{}, err
	}
	return u.repository.CreateProjectResource(ctx, ProjectResourceCreate{
		ID: id, WorkspaceID: strings.TrimSpace(workspaceID), ProjectID: strings.TrimSpace(projectID),
		ResourceType: string(kind), ResourceRef: canonical,
		Fingerprint: projectresource.Fingerprint(kind, reference), Label: label,
		IdempotencyKey: idempotencyKey, RequestHash: requestHash, Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectResourceUseCase) UpdateProjectResource(ctx context.Context, workspaceID, projectID, resourceID string, request contract.UpdateProjectResourceRequest) (contract.ProjectResource, error) {
	actor, _, err := u.authorizeProjectResourceManage(ctx, workspaceID, projectID)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	resourceID = strings.TrimSpace(resourceID)
	action := strings.TrimSpace(request.Action)
	if resourceID == "" || request.ExpectedRevision < 1 ||
		(action != "update" && action != "reorder" && action != "restore" && action != "refresh") {
		return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
	}
	command := ProjectResourceMutation{
		WorkspaceID: strings.TrimSpace(workspaceID), ProjectID: strings.TrimSpace(projectID),
		ResourceID: resourceID, Action: action, ExpectedRevision: request.ExpectedRevision,
		BeforeResourceID: normalizedResourceIDPointer(request.BeforeResourceID), Actor: actor, OccurredAt: u.now().UTC(),
	}
	if request.Label != nil {
		label := strings.TrimSpace(*request.Label)
		if len(label) > 120 {
			return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
		}
		command.Label = &label
	}
	switch action {
	case "update":
		if (request.ResourceRef == nil && request.Label == nil) || request.BeforeResourceID != nil {
			return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
		}
		if request.ResourceRef != nil {
			current, currentErr := u.repository.GetProjectResource(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), resourceID)
			if currentErr != nil {
				return contract.ProjectResource{}, currentErr
			}
			reference, normalizeErr := projectresource.Normalize(projectresource.Type(current.ResourceType), request.ResourceRef.URL, request.ResourceRef.Ref)
			if normalizeErr != nil {
				return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
			}
			canonical := contract.ProjectResourceRef{URL: reference.URL, Ref: reference.Ref}
			command.ResourceRef = &canonical
			command.Fingerprint = projectresource.Fingerprint(projectresource.Type(current.ResourceType), reference)
		}
	case "reorder":
		if request.ResourceRef != nil || request.Label != nil {
			return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
		}
	case "restore":
		if request.ResourceRef != nil || request.Label != nil || request.BeforeResourceID != nil {
			return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
		}
	case "refresh":
		if request.ResourceRef != nil || request.Label != nil || request.BeforeResourceID != nil {
			return contract.ProjectResource{}, ErrInvalidProjectResourceRequest
		}
		return u.repository.RefreshProjectResource(ctx, command, func(checkContext context.Context, checkRequest contract.ProjectResourceConnectionRequest) contract.ProjectResourceConnection {
			connection, checkErr := u.checker.Check(checkContext, checkRequest)
			return safeProjectResourceConnection(connection, checkErr, command.OccurredAt)
		})
	}
	return u.repository.MutateProjectResource(ctx, command)
}

func (u *ProjectResourceUseCase) ArchiveProjectResource(ctx context.Context, workspaceID, projectID, resourceID string, expectedRevision int64) error {
	actor, _, err := u.authorizeProjectResourceManage(ctx, workspaceID, projectID)
	if err != nil {
		return err
	}
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" || expectedRevision < 1 {
		return ErrInvalidProjectResourceRequest
	}
	return u.repository.ArchiveProjectResource(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), resourceID, expectedRevision, actor, u.now().UTC())
}

func (u *ProjectResourceUseCase) authorizeProjectResourceManage(ctx context.Context, workspaceID, projectID string) (contract.WorkspaceActor, ProjectResourceProjectAccess, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	if workspaceID == "" || projectID == "" {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, ErrInvalidProjectResourceRequest
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionResourceRead); err != nil {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, err
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, contract.ErrWorkspacePermissionDenied
	}
	membership, found, err := u.memberships.FindForUserAndWorkspace(ctx, actor.ID, workspaceID)
	if err == nil && !found {
		membership, found, err = u.memberships.FindByMemberAndWorkspace(ctx, actor.ID, workspaceID)
	}
	if err != nil {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, err
	}
	if !found {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, contract.ErrActorOutsideWorkspace
	}
	access, err := u.repository.ProjectResourceAccess(ctx, workspaceID, projectID)
	if err != nil {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, err
	}
	if access.Status == "completed" || access.Status == "cancelled" {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, contract.ErrWorkspacePermissionDenied
	}
	isLead := access.LeadType == "member" && (access.LeadID == actor.ID || access.LeadID == membership.MemberID || access.LeadID == membership.UserID)
	if membership.Role != "owner" && membership.Role != "admin" && !isLead {
		return contract.WorkspaceActor{}, ProjectResourceProjectAccess{}, contract.ErrWorkspacePermissionDenied
	}
	return actor, access, nil
}

func safeProjectResourceConnection(connection contract.ProjectResourceConnection, err error, checkedAt time.Time) contract.ProjectResourceConnection {
	if err != nil {
		return contract.ProjectResourceConnection{State: "unavailable", DiagnosticCode: "connection_check_failed", CheckedAt: checkedAt.Format(time.RFC3339Nano)}
	}
	if connection.State != "available" && connection.State != "degraded" && connection.State != "unavailable" {
		return contract.ProjectResourceConnection{State: "unavailable", DiagnosticCode: "invalid_connection_projection", CheckedAt: checkedAt.Format(time.RFC3339Nano)}
	}
	if connection.DiagnosticCode != "" && !projectResourceDiagnosticPattern.MatchString(connection.DiagnosticCode) {
		connection.DiagnosticCode = "invalid_connection_projection"
		connection.State = "unavailable"
	}
	connection.CheckedAt = checkedAt.Format(time.RFC3339Nano)
	return connection
}

func normalizedResourceIDPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func projectResourceRequestHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

var _ contract.ProjectResourceService = (*ProjectResourceUseCase)(nil)
