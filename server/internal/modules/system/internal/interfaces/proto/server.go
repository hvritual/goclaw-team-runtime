package proto

import (
	"context"
	systemv1 "github.com/multica-ai/multica/server/gen/go/system/v1"
	"github.com/multica-ai/multica/server/internal/modules/system/contract"
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
