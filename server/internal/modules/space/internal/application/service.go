package application

import "context"

type Service struct{}

func New() *Service                                   { return &Service{} }
func (*Service) Ping(context.Context) (string, error) { return "pong", nil }
