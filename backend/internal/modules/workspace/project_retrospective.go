package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type projectRetrospectiveExtension struct {
	handler *workspacehttp.ProjectRetrospectiveHandler
}

func newProjectRetrospectiveExtension(service contract.ProjectRetrospectiveService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *projectRetrospectiveExtension {
	return &projectRetrospectiveExtension{handler: workspacehttp.NewProjectRetrospectiveHandler(service, identity, authenticate, mutation)}
}

func (e *projectRetrospectiveExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}

func (e *projectRetrospectiveExtension) RegisterGRPC(grpc.ServiceRegistrar) {}
