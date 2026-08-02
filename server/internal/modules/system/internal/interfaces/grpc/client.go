package grpc

import (
	"context"

	systemv1 "github.com/multica-ai/multica/server/gen/go/system/v1"
	"github.com/multica-ai/multica/server/internal/modules/system/contract"
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
