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
	PermissionIssueMetadataGet    = "workspace.issue.metadata.get"
	PermissionIssueMetadataPut    = "workspace.issue.metadata.put"
	PermissionIssueMetadataDelete = "workspace.issue.metadata.delete"
)

type IssueMetadataRepository interface {
	GetMetadata(context.Context, string, string) (string, map[string]any, time.Time, error)
	PutMetadata(context.Context, string, string, string, any, time.Time) (string, map[string]any, time.Time, error)
	DeleteMetadata(context.Context, string, string, string, time.Time) (string, map[string]any, time.Time, error)
}

type IssueMetadataUseCase struct {
	repository IssueMetadataRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
	now        Clock
}

func NewIssueMetadataUseCase(repository IssueMetadataRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, now Clock) (*IssueMetadataUseCase, error) {
	if repository == nil || authorizer == nil || actors == nil || now == nil {
		return nil, errors.New("Issue metadata dependencies are required")
	}
	return &IssueMetadataUseCase{repository: repository, authorizer: authorizer, actors: actors, now: now}, nil
}

func (s *IssueMetadataUseCase) GetIssueMetadata(ctx context.Context, request contract.GetIssueMetadataRequest) (contract.IssueMetadataSnapshot, error) {
	workspaceID, issueID, err := validateIssueIdentity(request.WorkspaceId, request.IssueId)
	if err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueMetadataGet); err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	id, values, updated, err := s.repository.GetMetadata(ctx, workspaceID, issueID)
	return metadataResult(id, values, updated, err, "get")
}

func (s *IssueMetadataUseCase) PutIssueMetadata(ctx context.Context, request contract.PutIssueMetadataRequest) (contract.IssueMetadataSnapshot, error) {
	workspaceID, issueID, err := validateIssueIdentity(request.WorkspaceId, request.IssueId)
	if err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	if err := issueDomain.ValidateMetadataKey(request.Key); err != nil {
		return contract.IssueMetadataSnapshot{}, fmt.Errorf("%w: %v", contract.ErrInvalidIssueMetadata, err)
	}
	value, err := issueDomain.ParseMetadataValueJSON(request.ValueJson)
	if err != nil {
		return contract.IssueMetadataSnapshot{}, fmt.Errorf("%w: %v", contract.ErrInvalidIssueMetadata, err)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueMetadataPut); err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	if err := s.requireActor(ctx, workspaceID); err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	id, values, updated, err := s.repository.PutMetadata(ctx, workspaceID, issueID, request.Key, value, s.now())
	return metadataResult(id, values, updated, err, "put")
}

func (s *IssueMetadataUseCase) DeleteIssueMetadata(ctx context.Context, request contract.DeleteIssueMetadataRequest) (contract.IssueMetadataSnapshot, error) {
	workspaceID, issueID, err := validateIssueIdentity(request.WorkspaceId, request.IssueId)
	if err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	key := request.Key
	if err := issueDomain.ValidateMetadataKey(key); err != nil {
		return contract.IssueMetadataSnapshot{}, fmt.Errorf("%w: %v", contract.ErrInvalidIssueMetadata, err)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, PermissionIssueMetadataDelete); err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	if err := s.requireActor(ctx, workspaceID); err != nil {
		return contract.IssueMetadataSnapshot{}, err
	}
	id, values, updated, err := s.repository.DeleteMetadata(ctx, workspaceID, issueID, key, s.now())
	return metadataResult(id, values, updated, err, "delete")
}

func (s *IssueMetadataUseCase) requireActor(ctx context.Context, workspaceID string) error {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return contract.ErrWorkspaceActorRequired
	}
	if actor.Type != "member" && actor.Type != "agent" {
		return fmt.Errorf("%w: actor type must be member or agent", contract.ErrInvalidIssueMetadata)
	}
	exists, err := s.actors.ActorBelongsToWorkspace(ctx, workspaceID, actor.Type, actor.ID)
	if err != nil {
		return fmt.Errorf("verify Workspace actor: %w", err)
	}
	if !exists {
		return contract.ErrActorOutsideWorkspace
	}
	return nil
}

func metadataResult(id string, values map[string]any, updated time.Time, err error, operation string) (contract.IssueMetadataSnapshot, error) {
	if errors.Is(err, ErrIssueRecordNotFound) {
		return contract.IssueMetadataSnapshot{}, contract.ErrIssueNotFound
	}
	if err != nil {
		if strings.Contains(err.Error(), "metadata cannot exceed") || strings.Contains(err.Error(), "metadata exceeds") {
			return contract.IssueMetadataSnapshot{}, fmt.Errorf("%w: %v", contract.ErrInvalidIssueMetadata, err)
		}
		return contract.IssueMetadataSnapshot{}, fmt.Errorf("%s Issue metadata: %w", operation, err)
	}
	return contract.IssueMetadataSnapshot{IssueId: id, Metadata: issueDomain.NewMetadataBag(values).Snapshot(), UpdatedAt: updated.UTC().Format(time.RFC3339Nano)}, nil
}
