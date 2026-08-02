package space

import (
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	"github.com/multica-ai/multica/server/internal/modules/space/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/space/internal/interfaces/proto"
)

// NewAssetExtensionWithService assembles the generated local and transport
// adapters around an explicit application contract.
func NewAssetExtensionWithService(service contract.AssetService) *AssetExtension {
	client := local.NewAsset(service)
	return &AssetExtension{local: client, server: protoadapter.NewAssetServer(client)}
}
