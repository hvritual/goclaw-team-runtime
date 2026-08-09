package grpc

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacev1 "github.com/hvritual/workspace/rpc/pb/workspace/v1"
)

type Client struct {
	client workspacev1.WorkspaceServiceClient
}

func New(client workspacev1.WorkspaceServiceClient) *Client { return &Client{client: client} }
func (c *Client) Ping(ctx context.Context) (string, error) {
	response, err := c.client.Ping(ctx, &workspacev1.PingRequest{})
	if err != nil {
		return "", err
	}
	return response.GetMessage(), nil
}

var _ contract.Service = (*Client)(nil)
