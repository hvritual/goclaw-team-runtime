package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type projectRequirementExtension struct {
	handler *workspacehttp.ProjectRequirementHandler
}

func newProjectRequirementExtension(service contract.ProjectRequirementService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *projectRequirementExtension {
	return &projectRequirementExtension{handler: workspacehttp.NewProjectRequirementHandler(service, identity, authenticate, mutation)}
}

func (e *projectRequirementExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}

func (e *projectRequirementExtension) RegisterGRPC(grpc.ServiceRegistrar) {}
