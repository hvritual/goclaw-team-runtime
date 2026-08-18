package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/system/contract"
)

type SkillFiles struct {
	repository contract.SkillFileRepository
	catalog    contract.SkillCatalogService
	objects    spacecontract.SkillObjectService
	authorize  contract.SkillAccessAuthorizer
	now        func() time.Time
	newID      func() string
}

func NewSkillFiles(repository contract.SkillFileRepository, catalog contract.SkillCatalogService, objects spacecontract.SkillObjectService, authorize contract.SkillAccessAuthorizer) *SkillFiles {
	return &SkillFiles{repository: repository, catalog: catalog, objects: objects, authorize: authorize, now: time.Now, newID: uuid.NewString}
}

func (s *SkillFiles) List(ctx context.Context, identity contract.SkillIdentity, skillID, versionID string) ([]contract.SkillFileManifest, error) {
	entry, err := s.catalog.Get(ctx, identity, skillID, versionID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListFiles(ctx, identity, skillID, entry.VersionID)
}

func (s *SkillFiles) Read(ctx context.Context, identity contract.SkillIdentity, skillID, versionID, pathValue string) (contract.SkillFileManifest, []byte, error) {
	canonical, err := canonicalArchivePath(pathValue)
	if err != nil {
		return contract.SkillFileManifest{}, nil, contract.ErrInvalidSkill
	}
	files, err := s.List(ctx, identity, skillID, versionID)
	if err != nil {
		return contract.SkillFileManifest{}, nil, err
	}
	for _, file := range files {
		if file.Path != canonical {
			continue
		}
		value, reader, err := s.objects.Open(ctx, identity.WorkspaceID, file.SpaceObjectID)
		if errors.Is(err, spacecontract.ErrSkillObjectNotFound) {
			return contract.SkillFileManifest{}, nil, contract.ErrSkillNotFound
		}
		if err != nil {
			return contract.SkillFileManifest{}, nil, err
		}
		defer reader.Close()
		body, err := io.ReadAll(io.LimitReader(reader, SkillImportMaxFileBytes+1))
		checksum := fmt.Sprintf("%x", sha256.Sum256(body))
		if err != nil || int64(len(body)) != value.SizeBytes || value.Checksum != file.Checksum || checksum != file.Checksum {
			return contract.SkillFileManifest{}, nil, errors.New("Skill file object integrity check failed")
		}
		return file, body, nil
	}
	return contract.SkillFileManifest{}, nil, contract.ErrSkillNotFound
}

func (s *SkillFiles) Mutate(ctx context.Context, identity contract.SkillIdentity, skillID, mode, pathValue string, body []byte, expectedRevision int64) (contract.SkillCatalogEntry, error) {
	if expectedRevision <= 0 || (mode != "add" && mode != "replace") {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	if err := s.authorize(ctx, identity, contract.PermissionSkillImport); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	validated, err := ValidateSkillFile(pathValue, body)
	if err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	current, err := s.repository.ListFiles(ctx, identity, skillID, "")
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	found := false
	hasManifest := validated.Path == "SKILL.md"
	var replacedBytes int64
	var totalBytes int64
	for _, file := range current {
		hasManifest = hasManifest || file.Path == "SKILL.md"
		totalBytes += file.SizeBytes
		if file.Path == validated.Path {
			found = true
			replacedBytes = file.SizeBytes
		}
	}
	if !hasManifest {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	if mode == "add" && found {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAlreadyExists
	}
	if mode == "replace" && !found {
		return contract.SkillCatalogEntry{}, contract.ErrSkillNotFound
	}
	if mode == "add" && len(current) >= SkillImportMaxFiles || totalBytes-replacedBytes+validated.SizeBytes > SkillImportMaxTotalBytes {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	staged, err := s.objects.Stage(ctx, spacecontract.StageSkillObjectRequest{WorkspaceID: identity.WorkspaceID, MediaType: validated.MediaType, Content: body})
	if err != nil {
		return contract.SkillCatalogEntry{}, fmt.Errorf("stage Skill file: %w", err)
	}
	succeeded := false
	defer func() {
		if !succeeded {
			_ = s.objects.Discard(context.WithoutCancel(ctx), identity.WorkspaceID, staged.ID)
		}
	}()
	entry, err := s.repository.CreateFileVersion(ctx, identity, skillID, contract.SkillFileMutation{
		Path: validated.Path, ExpectedRevision: expectedRevision,
		Object: &contract.SkillFileManifest{ID: s.newID(), SpaceObjectID: staged.ID, MediaType: staged.MediaType, SizeBytes: staged.SizeBytes, Checksum: staged.Checksum},
	}, s.newID(), s.now().UTC(), func(promoteContext context.Context, executor contract.SkillCreateExecutor, objectID string) error {
		return s.objects.Promote(promoteContext, executor, identity.WorkspaceID, objectID)
	})
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	succeeded = true
	return entry, nil
}

func (s *SkillFiles) Delete(ctx context.Context, identity contract.SkillIdentity, skillID, pathValue string, expectedRevision int64) (contract.SkillCatalogEntry, error) {
	if expectedRevision <= 0 {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	if err := s.authorize(ctx, identity, contract.PermissionSkillImport); err != nil {
		return contract.SkillCatalogEntry{}, contract.ErrSkillAccessDenied
	}
	canonical, err := canonicalArchivePath(pathValue)
	if err != nil || canonical == "SKILL.md" {
		return contract.SkillCatalogEntry{}, contract.ErrInvalidSkill
	}
	current, err := s.repository.ListFiles(ctx, identity, skillID, "")
	if err != nil {
		return contract.SkillCatalogEntry{}, err
	}
	found := false
	for _, file := range current {
		found = found || file.Path == canonical
	}
	if !found {
		return contract.SkillCatalogEntry{}, contract.ErrSkillNotFound
	}
	return s.repository.CreateFileVersion(ctx, identity, skillID, contract.SkillFileMutation{Path: canonical, Delete: true, ExpectedRevision: expectedRevision}, s.newID(), s.now().UTC(), nil)
}

var _ contract.SkillFileService = (*SkillFiles)(nil)
