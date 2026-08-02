// dddgen:service-implementation AssetService; method bodies are user-owned.
package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/domain/asset"
)

var ErrAssetRecordNotFound = errors.New("asset record not found")

type AssetRepository interface {
	BeginUpload(context.Context, asset.UploadIntent) error
	FinalizeUpload(context.Context, asset.UploadIntent, string) (asset.Asset, error)
	FindByID(context.Context, string) (asset.Asset, error)
	ListPendingUploads(context.Context) ([]asset.UploadIntent, error)
	MarkCleanupPending(context.Context, string, string) error
	MarkCleaned(context.Context, string) error
}

type AssetServiceOption func(*AssetService)

type AssetService struct {
	assets     AssetRepository
	workspaces contract.WorkspaceAccess
	objects    contract.ObjectStore
	newID      func() (string, error)
	now        func() time.Time
}

func WithAssetRepository(repository AssetRepository) AssetServiceOption {
	return func(service *AssetService) { service.assets = repository }
}

func WithWorkspaceAccess(access contract.WorkspaceAccess) AssetServiceOption {
	return func(service *AssetService) { service.workspaces = access }
}

func WithObjectStore(objects contract.ObjectStore) AssetServiceOption {
	return func(service *AssetService) { service.objects = objects }
}

func WithAssetIDGenerator(generator func() (string, error)) AssetServiceOption {
	return func(service *AssetService) { service.newID = generator }
}

func WithAssetClock(now func() time.Time) AssetServiceOption {
	return func(service *AssetService) { service.now = now }
}

func NewAssetService(options ...AssetServiceOption) *AssetService {
	service := &AssetService{
		newID: func() (string, error) {
			value, err := uuid.NewV7()
			return value.String(), err
		},
		now: time.Now,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *AssetService) Available() bool {
	return s != nil && s.assets != nil && s.workspaces != nil &&
		s.objects != nil && s.objects.Available() && s.newID != nil && s.now != nil
}

func (s *AssetService) UploadAsset(
	ctx context.Context,
	request contract.Asset_UploadAssetRequest,
) (contract.Asset_UploadAssetResponse, error) {
	actorUserID, ok := contract.AssetActor(ctx)
	if !ok {
		return contract.Asset_UploadAssetResponse{}, contract.ErrAssetActorRequired
	}
	if !s.Available() {
		return contract.Asset_UploadAssetResponse{}, contract.ErrAssetStorageUnavailable
	}
	if request.Filename == "" || request.MediaType == "" {
		return contract.Asset_UploadAssetResponse{}, contract.ErrAssetInvalid
	}
	if request.WorkspaceId == "" {
		return s.uploadDirectObject(ctx, actorUserID, request)
	}

	allowed, err := s.workspaces.IsMember(ctx, actorUserID, request.WorkspaceId)
	if err != nil {
		return contract.Asset_UploadAssetResponse{}, err
	}
	if !allowed {
		return contract.Asset_UploadAssetResponse{}, contract.ErrAssetWorkspaceForbidden
	}
	// Each upload retries recovery of any earlier object whose metadata did not
	// finalize. A retry failure leaves its durable intent pending and must not
	// make this independent upload unsafe.
	_ = s.ReconcilePendingUploads(ctx)

	intent, err := s.newUploadIntent(actorUserID, request)
	if err != nil {
		return contract.Asset_UploadAssetResponse{}, err
	}
	if err := s.assets.BeginUpload(ctx, intent); err != nil {
		return contract.Asset_UploadAssetResponse{}, fmt.Errorf("begin asset upload: %w", err)
	}
	rawURL, err := s.objects.Upload(ctx, intent.ObjectKey, request.Content, request.MediaType, request.Filename)
	if err != nil {
		return contract.Asset_UploadAssetResponse{}, s.failAndCleanup(ctx, intent, err, contract.ErrAssetUploadFailed)
	}
	value, err := s.assets.FinalizeUpload(ctx, intent, rawURL)
	if err != nil {
		return contract.Asset_UploadAssetResponse{}, s.failAndCleanup(ctx, intent, err, contract.ErrAssetFinalizeFailed)
	}
	result := assetContract(value)
	return contract.Asset_UploadAssetResponse{Asset: &result, Filename: value.Filename()}, nil
}

func (s *AssetService) GetAsset(
	ctx context.Context,
	request contract.Asset_GetAssetRequest,
) (contract.Asset_GetAssetResponse, error) {
	actorUserID, ok := contract.AssetActor(ctx)
	if !ok {
		return contract.Asset_GetAssetResponse{}, contract.ErrAssetActorRequired
	}
	if s.assets == nil || s.workspaces == nil {
		return contract.Asset_GetAssetResponse{}, contract.ErrAssetNotImplemented
	}
	value, err := s.assets.FindByID(ctx, request.AssetId)
	if errors.Is(err, ErrAssetRecordNotFound) {
		return contract.Asset_GetAssetResponse{}, contract.ErrAssetNotFound
	}
	if err != nil {
		return contract.Asset_GetAssetResponse{}, err
	}
	allowed, err := s.workspaces.IsMember(ctx, actorUserID, value.WorkspaceID())
	if err != nil {
		return contract.Asset_GetAssetResponse{}, err
	}
	if !allowed {
		return contract.Asset_GetAssetResponse{}, contract.ErrAssetNotFound
	}
	result := assetContract(value)
	return contract.Asset_GetAssetResponse{Asset: &result}, nil
}

func (s *AssetService) ReconcilePendingUploads(ctx context.Context) error {
	if s == nil || s.assets == nil || s.objects == nil || !s.objects.Available() {
		return contract.ErrAssetStorageUnavailable
	}
	intents, err := s.assets.ListPendingUploads(ctx)
	if err != nil {
		return err
	}
	var cleanupErrors []error
	for _, intent := range intents {
		if err := s.cleanupIntent(ctx, intent, "reconcile incomplete upload"); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func (s *AssetService) newUploadIntent(
	actorUserID string,
	request contract.Asset_UploadAssetRequest,
) (asset.UploadIntent, error) {
	intentID, err := s.newID()
	if err != nil {
		return asset.UploadIntent{}, fmt.Errorf("generate upload intent id: %w", err)
	}
	assetID, err := s.newID()
	if err != nil {
		return asset.UploadIntent{}, fmt.Errorf("generate asset id: %w", err)
	}
	versionID, err := s.newID()
	if err != nil {
		return asset.UploadIntent{}, fmt.Errorf("generate asset version id: %w", err)
	}
	digest := sha256.Sum256(request.Content)
	return asset.NewUploadIntent(asset.UploadIntent{
		ID: intentID, AssetID: assetID, VersionID: versionID,
		WorkspaceID: request.WorkspaceId, UploaderType: asset.UploaderMember,
		UploaderID: actorUserID, Filename: request.Filename,
		ObjectKey: "workspaces/" + request.WorkspaceId + "/" + assetID + path.Ext(request.Filename),
		MediaType: request.MediaType, SizeBytes: int64(len(request.Content)),
		Checksum: "sha256:" + hex.EncodeToString(digest[:]), CreatedAt: s.now().UTC(),
	})
}

func (s *AssetService) uploadDirectObject(
	ctx context.Context,
	actorUserID string,
	request contract.Asset_UploadAssetRequest,
) (contract.Asset_UploadAssetResponse, error) {
	id, err := s.newID()
	if err != nil {
		return contract.Asset_UploadAssetResponse{}, fmt.Errorf("generate direct object id: %w", err)
	}
	key := "users/" + actorUserID + "/" + id + path.Ext(request.Filename)
	rawURL, err := s.objects.Upload(ctx, key, request.Content, request.MediaType, request.Filename)
	if err != nil {
		return contract.Asset_UploadAssetResponse{}, fmt.Errorf("%w: %w", contract.ErrAssetUploadFailed, err)
	}
	return contract.Asset_UploadAssetResponse{
		DirectObjectId: id, DirectUrl: rawURL, Filename: request.Filename,
	}, nil
}

func (s *AssetService) failAndCleanup(
	ctx context.Context,
	intent asset.UploadIntent,
	cause error,
	contractError error,
) error {
	cleanupErr := s.cleanupIntent(ctx, intent, cause.Error())
	return errors.Join(contractError, cause, cleanupErr)
}

func (s *AssetService) cleanupIntent(ctx context.Context, intent asset.UploadIntent, reason string) error {
	if err := s.assets.MarkCleanupPending(ctx, intent.ID, reason); err != nil {
		return fmt.Errorf("mark upload cleanup pending: %w", err)
	}
	if err := s.objects.DeleteObject(ctx, intent.ObjectKey); err != nil {
		return fmt.Errorf("delete incomplete upload object: %w", err)
	}
	if err := s.assets.MarkCleaned(ctx, intent.ID); err != nil {
		return fmt.Errorf("mark upload cleaned: %w", err)
	}
	return nil
}

func assetContract(value asset.Asset) contract.Asset_Asset {
	return contract.Asset_Asset{
		Id: value.ID(), WorkspaceId: value.WorkspaceID(), CurrentVersionId: value.CurrentVersionID(),
		Filename: value.Filename(), UploaderType: value.UploaderType(), UploaderId: value.UploaderID(),
		ObjectKey: value.ObjectKey(), Url: value.URL(), MediaType: value.MediaType(),
		SizeBytes: strconv.FormatInt(value.SizeBytes(), 10), Checksum: value.Checksum(),
		CreatedAt: value.CreatedAt().Format(time.RFC3339Nano),
	}
}
