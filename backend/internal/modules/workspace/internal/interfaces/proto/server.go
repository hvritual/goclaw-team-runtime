package proto

import (
	"context"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacev1 "github.com/hvritual/workspace/rpc/pb/workspace/v1"
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
