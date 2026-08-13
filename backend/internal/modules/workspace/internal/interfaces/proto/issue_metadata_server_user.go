package proto

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	workspacev1 "github.com/hvritual/workspace/rpc/pb/workspace/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type IssueMetadataServer struct {
	workspacev1.UnimplementedIssueMetadataServiceServer
	service contract.IssueMetadataService
}

func NewIssueMetadataServer(service contract.IssueMetadataService) *IssueMetadataServer {
	return &IssueMetadataServer{service: service}
}

func (s *IssueMetadataServer) GetIssueMetadata(ctx context.Context, request *workspacev1.GetIssueMetadataRequest) (*workspacev1.GetIssueMetadataResponse, error) {
	result, err := s.service.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: request.GetWorkspaceId(), IssueId: request.GetIssueId()})
	if err != nil {
		return nil, err
	}
	metadata, err := structpb.NewStruct(result.Metadata)
	if err != nil {
		return nil, err
	}
	return &workspacev1.GetIssueMetadataResponse{IssueId: result.IssueId, Metadata: metadata, UpdatedAt: result.UpdatedAt}, nil
}

func (s *IssueMetadataServer) PutIssueMetadataKey(ctx context.Context, request *workspacev1.PutIssueMetadataKeyRequest) (*workspacev1.PutIssueMetadataKeyResponse, error) {
	result, err := s.service.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: request.GetWorkspaceId(), IssueId: request.GetIssueId(), Key: request.GetKey(), ValueJson: request.GetValueJson()})
	if err != nil {
		return nil, err
	}
	metadata, err := structpb.NewStruct(result.Metadata)
	if err != nil {
		return nil, err
	}
	return &workspacev1.PutIssueMetadataKeyResponse{IssueId: result.IssueId, Metadata: metadata, UpdatedAt: result.UpdatedAt}, nil
}

func (s *IssueMetadataServer) DeleteIssueMetadataKey(ctx context.Context, request *workspacev1.DeleteIssueMetadataKeyRequest) (*workspacev1.DeleteIssueMetadataKeyResponse, error) {
	result, err := s.service.DeleteIssueMetadata(ctx, contract.DeleteIssueMetadataRequest{WorkspaceId: request.GetWorkspaceId(), IssueId: request.GetIssueId(), Key: request.GetKey()})
	if err != nil {
		return nil, err
	}
	metadata, err := structpb.NewStruct(result.Metadata)
	if err != nil {
		return nil, err
	}
	return &workspacev1.DeleteIssueMetadataKeyResponse{IssueId: result.IssueId, Metadata: metadata, UpdatedAt: result.UpdatedAt}, nil
}
