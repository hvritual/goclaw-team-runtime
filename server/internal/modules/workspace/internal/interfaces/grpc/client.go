package grpc

import (
	"context"

	workspacev1 "github.com/multica-ai/multica/server/gen/go/workspace/v1"
	"github.com/multica-ai/multica/server/internal/modules/workspace/contract"
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
