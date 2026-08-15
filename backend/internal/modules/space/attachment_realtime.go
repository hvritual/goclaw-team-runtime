package space

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/space/contract"
)

type publishingAttachmentService struct {
	contract.AttachmentService
	events contract.WorkspaceEventPublisher
}

func (s publishingAttachmentService) Upload(ctx context.Context, request contract.UploadAttachmentRequest) (contract.Attachment, error) {
	value, err := s.AttachmentService.Upload(ctx, request)
	if err == nil {
		s.publishIssueBag(ctx, value)
	}
	return value, err
}

func (s publishingAttachmentService) Delete(ctx context.Context, attachmentID string) error {
	before, err := s.AttachmentService.Get(ctx, attachmentID)
	if err != nil {
		return err
	}
	if err := s.AttachmentService.Delete(ctx, attachmentID); err != nil {
		return err
	}
	s.publishIssueBag(ctx, before)
	return nil
}

func (s publishingAttachmentService) publishIssueBag(ctx context.Context, value contract.Attachment) {
	if s.events == nil || value.IssueID == nil {
		return
	}
	attachments, err := s.AttachmentService.ListIssue(ctx, value.WorkspaceID, *value.IssueID)
	if err != nil {
		return
	}
	actor, _ := contract.AttachmentActorFromContext(ctx)
	s.events.Publish(value.WorkspaceID, "issue_attachments:changed", map[string]any{
		"issue_id": *value.IssueID, "attachments": attachments,
	}, actor.ID, actor.Type)
}

var _ contract.AttachmentService = publishingAttachmentService{}
