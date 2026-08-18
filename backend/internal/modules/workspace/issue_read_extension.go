package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type IssueReadExtension struct{ handler *httpadapter.IssueReadHandler }

func newIssueReadExtension(service contract.IssueMutationService, search contract.IssueSearchService, catalog contract.IssueCatalogService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error, createEnabled, attachmentsEnabled bool) *IssueReadExtension {
	return &IssueReadExtension{handler: httpadapter.NewIssueReadHandler(service, search, catalog, identity, authenticate, mutation, createEnabled, attachmentsEnabled)}
}

func (e *IssueReadExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *IssueReadExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
