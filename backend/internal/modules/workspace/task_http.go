package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type taskHTTPExtension struct {
	handler *workspacehttp.TaskHandler
}

func newTaskHTTPExtension(service contract.TodoService, promotion contract.TaskPromotionService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *taskHTTPExtension {
	return &taskHTTPExtension{handler: workspacehttp.NewTaskHandler(service, identity, authenticate, mutation).WithPromotion(promotion)}
}

func (e *taskHTTPExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (e *taskHTTPExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}
