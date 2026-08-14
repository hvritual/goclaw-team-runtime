package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type projectSurfaceExtension struct {
	handler *workspacehttp.ProjectSurfaceHandler
}

func newProjectSurfaceExtension(service contract.ProjectSurfaceService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *projectSurfaceExtension {
	return &projectSurfaceExtension{handler: workspacehttp.NewProjectSurfaceHandler(service, identity, authenticate, mutation)}
}
func (e *projectSurfaceExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *projectSurfaceExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
