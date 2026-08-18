package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type knowledgeReviewHTTPExtension struct {
	handler *workspacehttp.KnowledgeReviewHandler
}

func newKnowledgeReviewHTTPExtension(service contract.KnowledgeReviewService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *knowledgeReviewHTTPExtension {
	return &knowledgeReviewHTTPExtension{handler: workspacehttp.NewKnowledgeReviewHandler(service, identity, authenticate, mutation)}
}
func (e *knowledgeReviewHTTPExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}
func (e *knowledgeReviewHTTPExtension) RegisterGRPC(grpc.ServiceRegistrar) {}
