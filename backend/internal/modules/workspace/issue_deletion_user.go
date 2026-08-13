package workspace

import (
	"context"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type IssueDeletionExtension struct {
	local contract.IssueDeletionService
	http  *httpadapter.IssueDeletionHandler
}

func newIssueDeletionExtension(service contract.IssueDeletionService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *IssueDeletionExtension {
	return &IssueDeletionExtension{local: service, http: httpadapter.NewIssueDeletionHandler(service, identity, authenticate, mutation)}
}
func (e *IssueDeletionExtension) RegisterHTTP(server *kratoshttp.Server) { e.http.Register(server) }
func (e *IssueDeletionExtension) RegisterGRPC(grpc.ServiceRegistrar)     {}

type publishingIssueDeletionService struct {
	contract.IssueDeletionService
	events contract.WorkspaceEventPublisher
}

func (s publishingIssueDeletionService) DeleteIssue(ctx context.Context, request contract.DeleteIssueRequest) (contract.DeleteIssueResponse, error) {
	response, err := s.IssueDeletionService.DeleteIssue(ctx, request)
	if err == nil {
		actorID, actorType := realtimeActor(ctx)
		s.events.Publish(request.WorkspaceID, "issue:deleted", map[string]any{"issue_id": response.IssueID}, actorID, actorType)
	}
	return response, err
}
