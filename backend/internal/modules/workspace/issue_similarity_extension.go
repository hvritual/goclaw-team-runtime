package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type issueSimilarityExtension struct {
	handler *workspacehttp.IssueSimilarityHandler
}

func newIssueSimilarityExtension(service contract.IssueSimilarityService, issues contract.IssueMutationService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error)) *issueSimilarityExtension {
	return &issueSimilarityExtension{handler: workspacehttp.NewIssueSimilarityHandler(service, issues, identity, authenticate)}
}

func (e *issueSimilarityExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}

func (e *issueSimilarityExtension) RegisterGRPC(grpc.ServiceRegistrar) {}
