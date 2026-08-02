package grpc

import (
	"context"

	spacev1 "github.com/multica-ai/multica/server/gen/go/space/v1"
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
)

type Client struct{ client spacev1.SpaceServiceClient }

func New(client spacev1.SpaceServiceClient) *Client { return &Client{client: client} }
func (c *Client) Ping(ctx context.Context) (string, error) {
	response, err := c.client.Ping(ctx, &spacev1.PingRequest{})
	if err != nil {
		return "", err
	}
	return response.GetMessage(), nil
}

var _ contract.Service = (*Client)(nil)
