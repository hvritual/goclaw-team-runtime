package main

import (
	"context"
	"os"
	"testing"
	"time"

	workspacev1 "github.com/hvritual/workspace/rpc/pb/workspace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestLiveGRPCHealthAndWorkspacePing(t *testing.T) {
	address := os.Getenv("BACKEND_LIVE_GRPC_ADDR")
	if address == "" {
		t.Skip("live gRPC probe requires BACKEND_LIVE_GRPC_ADDR")
	}

	connection, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close gRPC connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	health, err := grpc_health_v1.NewHealthClient(connection).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("gRPC health check: %v", err)
	}
	if health.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("gRPC health status = %s, want SERVING", health.Status)
	}

	response, err := workspacev1.NewWorkspaceServiceClient(connection).Ping(ctx, &workspacev1.PingRequest{})
	if err != nil {
		t.Fatalf("Workspace Ping: %v", err)
	}
	if response.Message != "pong" {
		t.Fatalf("Workspace Ping message = %q, want pong", response.Message)
	}
}
