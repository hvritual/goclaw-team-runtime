package proto

import (
	"context"
	"github.com/hvritual/workspace/internal/modules/space/contract"
	spacev1 "github.com/hvritual/workspace/rpc/pb/space/v1"
)

type Server struct {
	spacev1.UnimplementedSpaceServiceServer
	service contract.Service
}

func New(service contract.Service) *Server { return &Server{service: service} }
func (s *Server) Ping(ctx context.Context, _ *spacev1.PingRequest) (*spacev1.PingResponse, error) {
	message, err := s.service.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &spacev1.PingResponse{Message: message}, nil
}
