package contract_test

import (
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/multica-ai/multica/server/internal/modules/space"
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type successfulAssetService struct {
	uploadRequest contract.Asset_UploadAssetRequest
	getRequest    contract.Asset_GetAssetRequest
}

func (s *successfulAssetService) UploadAsset(
	_ context.Context,
	request contract.Asset_UploadAssetRequest,
) (contract.Asset_UploadAssetResponse, error) {
	s.uploadRequest = request
	value := assetRoundTripValue(request.WorkspaceId, request.Filename)
	return contract.Asset_UploadAssetResponse{Asset: &value, Filename: request.Filename}, nil
}

func (s *successfulAssetService) GetAsset(
	_ context.Context,
	request contract.Asset_GetAssetRequest,
) (contract.Asset_GetAssetResponse, error) {
	s.getRequest = request
	value := assetRoundTripValue("workspace-1", "notes.txt")
	value.Id = request.AssetId
	return contract.Asset_GetAssetResponse{Asset: &value}, nil
}

func assetRoundTripValue(workspaceID, filename string) contract.Asset_Asset {
	return contract.Asset_Asset{
		Id: "asset-1", WorkspaceId: workspaceID, CurrentVersionId: "version-1",
		Filename: filename, UploaderType: "member", UploaderId: "user-1",
		ObjectKey: "workspaces/" + workspaceID + "/asset-1.txt",
		Url:       "/uploads/workspaces/" + workspaceID + "/asset-1.txt",
		MediaType: "text/plain", SizeBytes: "5", Checksum: "sha256:hello",
		CreatedAt: "2026-08-02T00:00:00Z",
	}
}

func TestSpaceAssetGRPCRoundTripsBytesAndInt64Fields(t *testing.T) {
	service := &successfulAssetService{}
	extension := space.NewAssetExtensionWithService(service)
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	extension.RegisterGRPC(grpcServer)
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(grpcServer.Stop)
	connection, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	client := space.NewAssetGRPCClient(connection)

	content := []byte("hello")
	uploaded, err := client.UploadAsset(t.Context(), contract.Asset_UploadAssetRequest{
		WorkspaceId: "workspace-1", Filename: "notes.txt", MediaType: "text/plain", Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(service.uploadRequest.Content, content) || service.uploadRequest.MediaType != "text/plain" {
		t.Fatalf("unexpected upload request: %+v", service.uploadRequest)
	}
	if uploaded.Asset == nil || uploaded.Asset.SizeBytes != "5" || uploaded.Asset.CurrentVersionId != "version-1" {
		t.Fatalf("unexpected upload response: %+v", uploaded)
	}

	got, err := client.GetAsset(t.Context(), contract.Asset_GetAssetRequest{AssetId: "asset-detail"})
	if err != nil {
		t.Fatal(err)
	}
	if service.getRequest.AssetId != "asset-detail" || got.Asset == nil || got.Asset.Id != "asset-detail" || got.Asset.SizeBytes != "5" {
		t.Fatalf("unexpected get round trip request=%+v response=%+v", service.getRequest, got)
	}
}
