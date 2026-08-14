package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type IssueCatalogExtension struct {
	handler *httpadapter.IssueCatalogHandler
}

func newIssueCatalogExtension(service contract.IssueCatalogService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *IssueCatalogExtension {
	return &IssueCatalogExtension{handler: httpadapter.NewIssueCatalogHandler(service, identity, authenticate, mutation)}
}

func (e *IssueCatalogExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *IssueCatalogExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
