package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type SkillCatalog struct {
	repository        contract.SkillCatalogRepository
	authorize         contract.SkillAccessAuthorizer
	preflight         contract.SkillVisibilityPreflight
	bind              contract.SkillVisibilityBinder
	resolveVisibility contract.SkillVisibilityResolver
	listVisibility    contract.SkillVisibilityLister
	now               func() time.Time
	newID             func() string
}

func NewSkillCatalog(repository contract.SkillCatalogRepository, authorize contract.SkillAccessAuthorizer, preflight contract.SkillVisibilityPreflight, bind contract.SkillVisibilityBinder, resolveVisibility contract.SkillVisibilityResolver, listVisibility contract.SkillVisibilityLister) *SkillCatalog {
	return &SkillCatalog{repository: repository, authorize: authorize, preflight: preflight, bind: bind, resolveVisibility: resolveVisibility, listVisibility: listVisibility, now: time.Now, newID: uuid.NewString}
}

func (s *SkillCatalog) Create(ctx context.Context, identity contract.SkillIdentity, request contract.CreateSkillCatalogRequest) (contract.SkillCatalogEntry, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)
	request.WorkspaceID, request.ActorType, request.ActorID = identity.WorkspaceID, identity.ActorType, identity.ActorID
	if request.Name == "" || identity.WorkspaceID == "" || identity.ActorType == "" || identity.ActorID == "" {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	if err := s.authorize(ctx, identity, contract.PermissionSkillCreate); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if request.Config == nil {
		request.Config = map[string]any{}
	}
	skillID, versionID := s.newID(), s.newID()
	if s.preflight == nil || s.bind == nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if err := s.preflight(ctx, identity, skillID, versionID); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	return s.repository.Create(ctx, request, skillID, versionID, s.now().UTC(), func(bindingContext context.Context, executor contract.SkillCreateExecutor) error {
		return s.bind(bindingContext, executor, identity, skillID, versionID)
	})
}

func (s *SkillCatalog) CreateVersion(ctx context.Context, identity contract.SkillIdentity, skillID string, request contract.UpdateSkillCatalogRequest) (contract.SkillCatalogEntry, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillVersion); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if request.ExpectedRevision <= 0 || strings.TrimSpace(skillID) == "" {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	if request.Name != nil {
		trimmed := strings.TrimSpace(*request.Name)
		if trimmed == "" {
			return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
		}
		request.Name = &trimmed
	}
	if request.Description != nil {
		trimmed := strings.TrimSpace(*request.Description)
		request.Description = &trimmed
	}
	request.RequireManifest = s.authorize(ctx, identity, contract.PermissionSkillImport) == nil
	return s.repository.CreateVersion(ctx, identity, skillID, request, s.newID(), s.now().UTC())
}

func (s *SkillCatalog) TransitionVersion(ctx context.Context, identity contract.SkillIdentity, skillID, versionID, transition string, expectedRevision int64) (contract.SkillCatalogEntry, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillVersion); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if transition != "publish" && transition != "deprecate" {
		return contract.SkillCatalogEntry{}, contract.ErrSkillTransition
	}
	return s.repository.TransitionVersion(ctx, identity, skillID, versionID, transition, expectedRevision, s.now().UTC())
}

func (s *SkillCatalog) Archive(ctx context.Context, identity contract.SkillIdentity, skillID string, expectedRevision int64) error {
	if err := s.authorize(ctx, identity, contract.PermissionSkillArchive); err != nil {
		return contract.ErrSkillAccessDenied
	}
	return s.repository.Archive(ctx, identity, skillID, expectedRevision, s.now().UTC())
}

func (s *SkillCatalog) Restore(ctx context.Context, identity contract.SkillIdentity, skillID string, expectedRevision int64) (contract.SkillCatalogEntry, error) {
	if err := s.authorize(ctx, identity, contract.PermissionSkillArchive); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	return s.repository.Restore(ctx, identity, skillID, expectedRevision, s.now().UTC())
}

func (s *SkillCatalog) Get(ctx context.Context, identity contract.SkillIdentity, skillID, versionID string) (contract.SkillCatalogEntry, error) {
	if s.authorize(ctx, identity, contract.PermissionSkillCreate) == nil {
		return s.repository.Get(ctx, identity, skillID, versionID, true)
	}
	if identity.ActorType != "agent" {
		if err := s.authorize(ctx, identity, contract.PermissionSkillRead); err != nil {
			return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
		}
	}
	if s.resolveVisibility == nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	reference, err := s.resolveVisibility(ctx, identity.WorkspaceID, skillID)
	if err != nil || !reference.Enabled || reference.VersionID == "" {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if versionID != "" && versionID != reference.VersionID {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	if identity.ActorType == "agent" && !containsActor(reference.AgentIDs, identity.ActorID) {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	value, err := s.repository.GetReferenced(ctx, reference.SkillID, reference.VersionID)
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	if !readableReferencedStatus(value.Status) {
		return contract.SkillCatalogEntry{}, contract.ErrSkillNotFound
	}
	return value, nil
}

func (s *SkillCatalog) List(ctx context.Context, identity contract.SkillIdentity) ([]contract.SkillCatalogEntry, error) {
	if s.authorize(ctx, identity, contract.PermissionSkillCreate) == nil {
		return s.repository.List(ctx, identity, true)
	}
	if identity.ActorType != "agent" {
		if err := s.authorize(ctx, identity, contract.PermissionSkillRead); err != nil {
			return nil, contract.ErrSkillAccessDenied
		}
	}
	if s.listVisibility == nil {
		return nil, contract.ErrSkillAccessDenied
	}
	references, err := s.listVisibility(ctx, identity.WorkspaceID)
	if err != nil {
		return nil, err
	}
	values := make([]contract.SkillCatalogEntry, 0, len(references))
	for _, reference := range references {
		if !reference.Enabled || reference.VersionID == "" || identity.ActorType == "agent" && !containsActor(reference.AgentIDs, identity.ActorID) {
			continue
		}
		value, readErr := s.repository.GetReferenced(ctx, reference.SkillID, reference.VersionID)
		if errors.Is(readErr, contract.ErrSkillNotFound) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}
		if !value.Archived && readableReferencedStatus(value.Status) {
			values = append(values, value)
		}
	}
	return values, nil
}

func (s *SkillCatalog) History(ctx context.Context, identity contract.SkillIdentity, skillID string) (contract.SkillHistory, error) {
	if strings.TrimSpace(skillID) == "" {
		return contract.SkillHistory{}, contract.ErrInvalidSkill
	}
	if err := s.authorize(ctx, identity, contract.PermissionSkillCreate); err != nil {
		return contract.SkillHistory{}, contract.ErrSkillAccessDenied
	}
	return s.repository.History(ctx, identity, skillID)
}

func containsActor(values []string, actorID string) bool {
	for _, value := range values {
		if value == actorID {
			return true
		}
	}
	return false
}

func readableReferencedStatus(status string) bool {
	return status == "published" || status == "deprecated" || status == "archived"
}

var _ contract.SkillCatalogService = (*SkillCatalog)(nil)
