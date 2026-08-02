package proto

import (
	"context"
	spacev1 "github.com/multica-ai/multica/server/gen/go/space/v1"
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
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
