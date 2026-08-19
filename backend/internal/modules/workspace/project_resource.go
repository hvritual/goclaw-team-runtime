package workspace

import (
	"context"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type projectResourceExtension struct {
	handler *workspacehttp.ProjectResourceHandler
}

func newProjectResourceExtension(service contract.ProjectResourceService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *projectResourceExtension {
	return &projectResourceExtension{handler: workspacehttp.NewProjectResourceHandler(service, identity, authenticate, mutation)}
}

func (e *projectResourceExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}
func (e *projectResourceExtension) RegisterGRPC(grpc.ServiceRegistrar) {}

type unavailableProjectResourceConnectionChecker struct{}

func (unavailableProjectResourceConnectionChecker) Check(context.Context, contract.ProjectResourceConnectionRequest) (contract.ProjectResourceConnection, error) {
	return contract.ProjectResourceConnection{State: "unavailable", DiagnosticCode: "connection_not_configured"}, nil
}
