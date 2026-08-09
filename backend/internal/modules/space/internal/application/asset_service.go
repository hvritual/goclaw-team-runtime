// dddgen:service-implementation AssetService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/space/contract"
)

type AssetService struct{}

func NewAssetService() *AssetService { return &AssetService{} }

func (s *AssetService) UploadAsset(ctx context.Context, request contract.UploadAssetRequest) (contract.UploadAssetResponse, error) {
	return contract.UploadAssetResponse{}, contract.ErrAssetNotImplemented
}

func (s *AssetService) CreateAssetVersion(ctx context.Context, request contract.CreateAssetVersionRequest) (contract.CreateAssetVersionResponse, error) {
	return contract.CreateAssetVersionResponse{}, contract.ErrAssetNotImplemented
}

func (s *AssetService) GetAsset(ctx context.Context, request contract.GetAssetRequest) (contract.GetAssetResponse, error) {
	return contract.GetAssetResponse{}, contract.ErrAssetNotImplemented
}
