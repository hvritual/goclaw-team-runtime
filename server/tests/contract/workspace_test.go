package contract_test

import (
	"context"
	"net"
	"testing"

	"github.com/multica-ai/multica/server/internal/modules/workspace"
	"github.com/multica-ai/multica/server/internal/modules/workspace/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestWorkspaceAdaptersShareContract(t *testing.T) {
	module := workspace.New()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	module.RegisterGRPCService(server)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	connection, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = connection.Close() }()
	for name, client := range map[string]contract.Service{"local": module.Local(), "grpc": workspace.NewGRPCClient(connection)} {
		t.Run(name, func(t *testing.T) {
			message, callErr := client.Ping(context.Background())
			if callErr != nil || message != "pong" {
				t.Fatalf("Ping() = %q, %v", message, callErr)
			}
		})
	}
}
