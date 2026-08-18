package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type knowledgeQueryHTTPExtension struct {
	handler *workspacehttp.KnowledgeQueryHandler
}

func newKnowledgeQueryHTTPExtension(service contract.KnowledgeQueryService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error)) *knowledgeQueryHTTPExtension {
	return &knowledgeQueryHTTPExtension{handler: workspacehttp.NewKnowledgeQueryHandler(service, identity, authenticate)}
}

func (e *knowledgeQueryHTTPExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}
func (e *knowledgeQueryHTTPExtension) RegisterGRPC(grpc.ServiceRegistrar) {}
