package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	knowledgeDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/knowledge"
)

const (
	PermissionKnowledgeCreate = "workspace.knowledge.create"
	PermissionKnowledgeGet    = "workspace.knowledge.get"
)

var ErrKnowledgeRecordNotFound = errors.New("knowledge record not found")

type KnowledgeRepository interface {
	Create(context.Context, knowledgeDomain.Knowledge) error
	FindByID(context.Context, string, string) (knowledgeDomain.Knowledge, error)
}
type KnowledgeUseCase struct {
	repository KnowledgeRepository
	authorizer contract.WorkspaceAccessAuthorizer
	assets     contract.WorkspaceAssetReader
	newID      ProjectIDGenerator
	now        Clock
}

func NewKnowledgeUseCase(repository KnowledgeRepository, authorizer contract.WorkspaceAccessAuthorizer, assets contract.WorkspaceAssetReader, newID ProjectIDGenerator, now Clock) (*KnowledgeUseCase, error) {
	if repository == nil || authorizer == nil || assets == nil || newID == nil || now == nil {
		return nil, errors.New("Knowledge dependencies are required")
	}
	return &KnowledgeUseCase{repository: repository, authorizer: authorizer, assets: assets, newID: newID, now: now}, nil
}
func (s *KnowledgeUseCase) CreateKnowledge(ctx context.Context, r contract.CreateKnowledgeRequest) (contract.CreateKnowledgeResponse, error) {
	w := strings.TrimSpace(r.WorkspaceId)
	if w == "" {
		return contract.CreateKnowledgeResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidKnowledge)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, PermissionKnowledgeCreate); err != nil {
		return contract.CreateKnowledgeResponse{}, err
	}
	id, err := s.newID(ctx)
	if err != nil {
		return contract.CreateKnowledgeResponse{}, fmt.Errorf("generate Knowledge id: %w", err)
	}
	v, err := knowledgeDomain.New(id, w, r.Title, r.Summary, r.AssetIds, s.now())
	if err != nil {
		return contract.CreateKnowledgeResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidKnowledge, err)
	}
	for _, asset := range v.AssetIDs {
		ok, err := s.assets.AssetBelongsToWorkspace(ctx, w, asset)
		if err != nil {
			return contract.CreateKnowledgeResponse{}, fmt.Errorf("validate Knowledge Asset: %w", err)
		}
		if !ok {
			return contract.CreateKnowledgeResponse{}, contract.ErrAssetOutsideWorkspace
		}
	}
	if err := s.repository.Create(ctx, v); err != nil {
		return contract.CreateKnowledgeResponse{}, fmt.Errorf("create Knowledge: %w", err)
	}
	out := knowledgeToContract(v)
	return contract.CreateKnowledgeResponse{Knowledge: &out}, nil
}
func (s *KnowledgeUseCase) GetKnowledge(ctx context.Context, r contract.GetKnowledgeRequest) (contract.GetKnowledgeResponse, error) {
	w, id := strings.TrimSpace(r.WorkspaceId), strings.TrimSpace(r.KnowledgeId)
	if w == "" || id == "" {
		return contract.GetKnowledgeResponse{}, fmt.Errorf("%w: workspace id and Knowledge id are required", contract.ErrInvalidKnowledge)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, PermissionKnowledgeGet); err != nil {
		return contract.GetKnowledgeResponse{}, err
	}
	v, err := s.repository.FindByID(ctx, w, id)
	if errors.Is(err, ErrKnowledgeRecordNotFound) {
		return contract.GetKnowledgeResponse{}, contract.ErrKnowledgeNotFound
	}
	if err != nil {
		return contract.GetKnowledgeResponse{}, fmt.Errorf("get Knowledge: %w", err)
	}
	out := knowledgeToContract(v)
	return contract.GetKnowledgeResponse{Knowledge: &out}, nil
}
func knowledgeToContract(v knowledgeDomain.Knowledge) contract.Knowledge {
	return contract.Knowledge{Id: v.ID, WorkspaceId: v.WorkspaceID, Title: v.Title, Summary: v.Summary, Status: v.Status, AssetIds: append([]string(nil), v.AssetIDs...), CreatedAt: v.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: v.UpdatedAt.Format(time.RFC3339Nano)}
}

var _ contract.KnowledgeService = (*KnowledgeUseCase)(nil)
