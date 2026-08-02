package proto

import (
	"context"
	workspacev1 "github.com/multica-ai/multica/server/gen/go/workspace/v1"
	"github.com/multica-ai/multica/server/internal/modules/workspace/contract"
)

type Server struct {
	workspacev1.UnimplementedWorkspaceServiceServer
	service contract.Service
}

func New(service contract.Service) *Server { return &Server{service: service} }
func (s *Server) Ping(ctx context.Context, _ *workspacev1.PingRequest) (*workspacev1.PingResponse, error) {
	message, err := s.service.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &workspacev1.PingResponse{Message: message}, nil
}
