package proto

import (
	"context"
	"github.com/hvritual/workspace/internal/modules/system/contract"
	systemv1 "github.com/hvritual/workspace/rpc/pb/system/v1"
)

type Server struct {
	systemv1.UnimplementedSystemServiceServer
	service contract.Service
}

func New(service contract.Service) *Server { return &Server{service: service} }
func (s *Server) Ping(ctx context.Context, _ *systemv1.PingRequest) (*systemv1.PingResponse, error) {
	message, err := s.service.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &systemv1.PingResponse{Message: message}, nil
}
