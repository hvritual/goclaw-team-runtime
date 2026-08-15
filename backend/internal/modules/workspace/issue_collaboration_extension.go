package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type IssueCollaborationExtension struct {
	handler *httpadapter.IssueCollaborationHandler
}

func newIssueCollaborationExtension(service contract.IssueCollaborationService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error, attachmentsEnabled bool) *IssueCollaborationExtension {
	return &IssueCollaborationExtension{handler: httpadapter.NewIssueCollaborationHandler(service, identity, authenticate, mutation, attachmentsEnabled)}
}

func (e *IssueCollaborationExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}
func (e *IssueCollaborationExtension) RegisterGRPC(grpc.ServiceRegistrar) {}
