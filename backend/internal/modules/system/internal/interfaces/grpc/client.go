package grpc

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/system/contract"
	systemv1 "github.com/hvritual/workspace/rpc/pb/system/v1"
)

type Client struct{ client systemv1.SystemServiceClient }

func New(client systemv1.SystemServiceClient) *Client { return &Client{client: client} }
func (c *Client) Ping(ctx context.Context) (string, error) {
	response, err := c.client.Ping(ctx, &systemv1.PingRequest{})
	if err != nil {
		return "", err
	}
	return response.GetMessage(), nil
}

var _ contract.Service = (*Client)(nil)
