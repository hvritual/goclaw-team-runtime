package contract

import (
	"context"
	"errors"
	"io"
)

var (
	ErrAssetActorRequired      = errors.New("asset actor is required")
	ErrAssetStorageUnavailable = errors.New("file upload not configured")
	ErrAssetWorkspaceForbidden = errors.New("not a member of this workspace")
	ErrAssetInvalid            = errors.New("invalid asset upload")
	ErrAssetUploadFailed       = errors.New("upload failed")
	ErrAssetFinalizeFailed     = errors.New("failed to finalize asset")
	ErrAssetNotFound           = errors.New("attachment not found")
)

type assetActorContextKey struct{}

// WithAssetActor attaches the authenticated user identity to an AssetService call.
func WithAssetActor(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, assetActorContextKey{}, userID)
}

// AssetActor returns the authenticated user identity for an AssetService call.
func AssetActor(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(assetActorContextKey{}).(string)
	return userID, ok && userID != ""
}

// WorkspaceAccess is the consumer-owned authorization port supplied by Auth composition.
type WorkspaceAccess interface {
	IsMember(ctx context.Context, userID, workspaceID string) (bool, error)
}

// ObjectStore is the Space-owned object lifecycle port implemented by local or remote storage.
type ObjectStore interface {
	Available() bool
	Upload(ctx context.Context, key string, data []byte, mediaType, filename string) (string, error)
	DeleteObject(ctx context.Context, key string) error
	KeyFromURL(rawURL string) string
	GetReader(ctx context.Context, key string) (io.ReadCloser, error)
}

// AssetUploadService exposes the generated contract plus the operational seams
// needed by the installed multipart and consumer-relation adapters.
type AssetUploadService interface {
	AssetService
	Available() bool
	ReconcilePendingUploads(context.Context) error
}
