package local

import (
	"context"
	"github.com/hvritual/workspace/internal/modules/space/contract"
)

type Client struct{ service contract.Service }

func New(service contract.Service) *Client                 { return &Client{service: service} }
func (c *Client) Ping(ctx context.Context) (string, error) { return c.service.Ping(ctx) }
