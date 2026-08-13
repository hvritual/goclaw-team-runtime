package workspace

import (
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type IssueReadExtension struct{ handler *httpadapter.IssueReadHandler }

func newIssueReadExtension(service contract.IssueService, identity contract.WorkspaceHTTPIdentityResolver) *IssueReadExtension {
	return &IssueReadExtension{handler: httpadapter.NewIssueReadHandler(service, identity)}
}

func (e *IssueReadExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *IssueReadExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
