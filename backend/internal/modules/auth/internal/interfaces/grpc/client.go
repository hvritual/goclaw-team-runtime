package grpc

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
	authv1 "github.com/hvritual/workspace/rpc/pb/auth/v1"
)

type Client struct{ client authv1.AuthServiceClient }

func New(client authv1.AuthServiceClient) *Client { return &Client{client: client} }
func (c *Client) Ping(ctx context.Context) (string, error) {
	response, err := c.client.Ping(ctx, &authv1.PingRequest{})
	if err != nil {
		return "", err
	}
	return response.GetMessage(), nil
}

var _ contract.Service = (*Client)(nil)
