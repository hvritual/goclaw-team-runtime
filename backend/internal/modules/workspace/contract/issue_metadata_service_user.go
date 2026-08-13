package contract

import "context"

type GetIssueMetadataRequest struct {
	WorkspaceId string
	IssueId     string
}

type PutIssueMetadataRequest struct {
	WorkspaceId string
	IssueId     string
	Key         string
	ValueJson   string
}

type DeleteIssueMetadataRequest struct {
	WorkspaceId string
	IssueId     string
	Key         string
}

type IssueMetadataSnapshot struct {
	IssueId   string
	Metadata  map[string]any
	UpdatedAt string
}

type IssueMetadataService interface {
	GetIssueMetadata(context.Context, GetIssueMetadataRequest) (IssueMetadataSnapshot, error)
	PutIssueMetadata(context.Context, PutIssueMetadataRequest) (IssueMetadataSnapshot, error)
	DeleteIssueMetadata(context.Context, DeleteIssueMetadataRequest) (IssueMetadataSnapshot, error)
}
