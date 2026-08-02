// Package spaceacl maps consumer-owned Issue upload contracts to Space.
package spaceacl

import (
	"context"
	"errors"

	"github.com/multica-ai/multica/server/internal/issueguard"
	spaceapp "github.com/multica-ai/multica/server/modules/space/application"
	"github.com/multica-ai/multica/server/modules/space/domain"
)

// Provider adapts Space application use cases to the Issue-owned provider port.
type Provider struct {
	service *spaceapp.UploadService
}

// NewProvider creates the Issue-to-Space anti-corruption adapter.
func NewProvider(service *spaceapp.UploadService) *Provider {
	return &Provider{service: service}
}

// Available reports whether the Space upload provider is configured.
func (a *Provider) Available() bool {
	return a != nil && a.service != nil && a.service.Available()
}

// Upload maps an ordinary consumer upload to Space.
func (a *Provider) Upload(ctx context.Context, command issueguard.SpaceUploadCommand) (issueguard.SpaceUploadResult, error) {
	result, err := a.service.Upload(ctx, spaceapp.UploadCommand{
		UserID:      command.UserID,
		WorkspaceID: command.WorkspaceID,
		Filename:    command.Filename,
		ContentType: command.ContentType,
		Content:     command.Content,
	})
	if err != nil {
		return issueguard.SpaceUploadResult{}, mapProviderError(err)
	}
	return issueguard.SpaceUploadResult{
		Asset:    assetReference(result.Asset),
		ID:       result.ID,
		URL:      result.URL,
		Filename: result.Filename,
	}, nil
}

// PrepareWorkspaceAsset maps the staged Space Asset to the Issue-owned DTO.
func (a *Provider) PrepareWorkspaceAsset(ctx context.Context, command issueguard.SpaceUploadCommand) (issueguard.PreparedAsset, error) {
	asset, err := a.service.PrepareWorkspaceAsset(ctx, spaceapp.UploadCommand{
		UserID:      command.UserID,
		WorkspaceID: command.WorkspaceID,
		Filename:    command.Filename,
		ContentType: command.ContentType,
		Content:     command.Content,
	})
	if err != nil {
		return issueguard.PreparedAsset{}, mapProviderError(err)
	}
	return *assetReference(&asset), nil
}

func mapProviderError(err error) error {
	var metadataErr *spaceapp.MetadataPersistenceError
	if errors.As(err, &metadataErr) {
		return &issueguard.MetadataPersistenceError{
			Result: issueguard.UploadResult{
				ID:       metadataErr.Result.ID,
				URL:      metadataErr.Result.URL,
				Filename: metadataErr.Result.Filename,
			},
			Err: metadataErr.Err,
		}
	}
	return joinMappedError(err, []errorMapping{
		{spaceapp.ErrStorageUnavailable, issueguard.ErrStorageUnavailable},
		{spaceapp.ErrNotWorkspaceMember, issueguard.ErrNotWorkspaceMember},
		{spaceapp.ErrUploadFailed, issueguard.ErrUploadFailed},
		{spaceapp.ErrGenerateID, issueguard.ErrGenerateID},
	})
}

type errorMapping struct {
	from error
	to   error
}

func joinMappedError(err error, mappings []errorMapping) error {
	for _, mapping := range mappings {
		if errors.Is(err, mapping.from) {
			return errors.Join(mapping.to, err)
		}
	}
	return err
}

func assetReference(asset *domain.Asset) *issueguard.PreparedAsset {
	if asset == nil {
		return nil
	}
	return &issueguard.PreparedAsset{
		ID:           asset.ID(),
		WorkspaceID:  asset.WorkspaceID(),
		UploaderType: string(asset.UploaderType()),
		UploaderID:   asset.UploaderID(),
		Filename:     asset.Filename(),
		URL:          asset.URL(),
		ContentType:  asset.ContentType(),
		SizeBytes:    asset.SizeBytes(),
		Checksum:     asset.Checksum(),
		CreatedAt:    asset.CreatedAt(),
	}
}
