// Package application coordinates Space use cases through inward-facing ports.
package application

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/multica-ai/multica/server/modules/space/domain"
)

var (
	// ErrStorageUnavailable reports that no object storage provider is configured.
	ErrStorageUnavailable = errors.New("file upload not configured")
	// ErrNotWorkspaceMember reports a workspace authorization failure.
	ErrNotWorkspaceMember = errors.New("not a member of this workspace")
	// ErrUploadFailed reports a failure to write the object body.
	ErrUploadFailed = errors.New("upload failed")
	// ErrGenerateID reports a failure to allocate an Asset identity.
	ErrGenerateID = errors.New("generate asset id")
)

// AssetRepository persists the workspace-scoped metadata supported by the
// installed attachment schema. The computed checksum remains part of the
// upload result until the planned Asset-owned schema can store it durably.
type AssetRepository interface {
	Create(ctx context.Context, asset domain.Asset) (domain.Asset, error)
}

// WorkspaceAccess authorizes a user against the workspace boundary.
type WorkspaceAccess interface {
	IsMember(ctx context.Context, userID, workspaceID string) bool
}

// ObjectStore persists file bytes without exposing provider types inward.
type ObjectStore interface {
	Available() bool
	Upload(ctx context.Context, key string, data []byte, contentType, filename string) (string, error)
}

// IDGenerator allocates stable Asset identities.
type IDGenerator interface {
	NewID() (string, error)
}

// Checksummer computes the immutable digest recorded by the Asset domain.
type Checksummer interface {
	Sum(data []byte) string
}

// UploadCommand contains validated HTTP upload input.
type UploadCommand struct {
	UserID      string
	WorkspaceID string
	Filename    string
	ContentType string
	Content     []byte
}

// UploadResult describes either a persisted Asset or a legacy personal object.
type UploadResult struct {
	Asset    *domain.Asset
	ID       string
	URL      string
	Filename string
}

// MetadataPersistenceError carries the legacy direct-link fallback when object
// storage succeeded but metadata persistence failed.
type MetadataPersistenceError struct {
	Result UploadResult
	Err    error
}

func (e *MetadataPersistenceError) Error() string { return "persist asset metadata: " + e.Err.Error() }
func (e *MetadataPersistenceError) Unwrap() error { return e.Err }

// UploadService implements the upload and finalize Asset tracer slice.
type UploadService struct {
	assets     AssetRepository
	workspaces WorkspaceAccess
	objects    ObjectStore
	ids        IDGenerator
	checksums  Checksummer
}

// NewUploadService binds the upload use case to consumer-owned ports.
func NewUploadService(
	assets AssetRepository,
	workspaces WorkspaceAccess,
	objects ObjectStore,
	ids IDGenerator,
	checksums Checksummer,
) *UploadService {
	return &UploadService{
		assets:     assets,
		workspaces: workspaces,
		objects:    objects,
		ids:        ids,
		checksums:  checksums,
	}
}

// Available reports whether the configured object-store adapter can upload.
func (s *UploadService) Available() bool {
	return s != nil && s.objects != nil && s.objects.Available()
}

// Upload stores file content and, for workspace uploads, persists Asset metadata.
func (s *UploadService) Upload(ctx context.Context, command UploadCommand) (UploadResult, error) {
	if !s.Available() {
		return UploadResult{}, ErrStorageUnavailable
	}

	if command.WorkspaceID != "" {
		asset, err := s.PrepareWorkspaceAsset(ctx, command)
		if err != nil {
			return UploadResult{}, err
		}
		stored, err := s.assets.Create(ctx, asset)
		if err != nil {
			return UploadResult{}, newMetadataPersistenceError(asset, err)
		}
		return UploadResult{Asset: &stored}, nil
	}
	id, err := s.newID()
	if err != nil {
		return UploadResult{}, err
	}
	return s.uploadPersonalObject(ctx, id, command)
}

// PrepareWorkspaceAsset authorizes and stores a workspace object without
// committing its metadata. Consumer contexts use this seam when they must add
// their own relation in the same metadata insert.
func (s *UploadService) PrepareWorkspaceAsset(ctx context.Context, command UploadCommand) (domain.Asset, error) {
	if !s.Available() {
		return domain.Asset{}, ErrStorageUnavailable
	}
	if !s.workspaces.IsMember(ctx, command.UserID, command.WorkspaceID) {
		return domain.Asset{}, ErrNotWorkspaceMember
	}
	id, err := s.newID()
	if err != nil {
		return domain.Asset{}, err
	}
	key := "workspaces/" + command.WorkspaceID + "/" + id + path.Ext(command.Filename)
	checksum := s.checksums.Sum(command.Content)
	url, err := s.objects.Upload(ctx, key, command.Content, command.ContentType, command.Filename)
	if err != nil {
		return domain.Asset{}, fmt.Errorf("%w: %w", ErrUploadFailed, err)
	}

	asset, err := domain.NewUploadedAsset(domain.UploadedAssetParams{
		ID:           id,
		WorkspaceID:  command.WorkspaceID,
		UploaderType: domain.UploaderMember,
		UploaderID:   command.UserID,
		Filename:     command.Filename,
		URL:          url,
		ContentType:  command.ContentType,
		SizeBytes:    int64(len(command.Content)),
		Checksum:     checksum,
	})
	if err != nil {
		return domain.Asset{}, fmt.Errorf("build uploaded asset: %w", err)
	}
	return asset, nil
}

func (s *UploadService) newID() (string, error) {
	id, err := s.ids.NewID()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGenerateID, err)
	}
	return id, nil
}

func newMetadataPersistenceError(asset domain.Asset, err error) *MetadataPersistenceError {
	// Preserve the installed-client compatibility behavior of the legacy
	// endpoint: if object storage succeeded but the metadata insert failed,
	// return the direct link with an empty attachment id.
	return &MetadataPersistenceError{
		Result: UploadResult{URL: asset.URL(), Filename: asset.Filename()},
		Err:    err,
	}
}

func (s *UploadService) uploadPersonalObject(ctx context.Context, id string, command UploadCommand) (UploadResult, error) {
	key := "users/" + command.UserID + "/" + id + path.Ext(command.Filename)
	url, err := s.objects.Upload(ctx, key, command.Content, command.ContentType, command.Filename)
	if err != nil {
		return UploadResult{}, fmt.Errorf("%w: %w", ErrUploadFailed, err)
	}
	return UploadResult{ID: id, URL: url, Filename: command.Filename}, nil
}
