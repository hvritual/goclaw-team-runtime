package space

import (
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/space/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/space/internal/interfaces/http"
	"google.golang.org/grpc"
)

type attachmentExtension struct {
	handler *httpadapter.AttachmentHandler
}

func newAttachmentExtension(service contract.AttachmentService, identity contract.HTTPIdentityResolver, user contract.HTTPUserResolver, mutation contract.HTTPMutationAuthorizer, memberships contract.WorkspaceMembershipReader) *attachmentExtension {
	return &attachmentExtension{handler: httpadapter.NewAttachmentHandler(service, identity, user, mutation, memberships)}
}

func (e *attachmentExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *attachmentExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
