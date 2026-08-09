package proto

import (
	"context"
	"github.com/hvritual/workspace/internal/modules/auth/contract"
	authv1 "github.com/hvritual/workspace/rpc/pb/auth/v1"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	service contract.Service
}

func New(service contract.Service) *Server { return &Server{service: service} }
func (s *Server) Ping(ctx context.Context, _ *authv1.PingRequest) (*authv1.PingResponse, error) {
	message, err := s.service.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &authv1.PingResponse{Message: message}, nil
}
