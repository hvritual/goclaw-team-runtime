// Package uploadhttp maps the cross-context upload workflow to its HTTP boundary.
package uploadhttp

import (
	"context"
	"errors"

	"github.com/multica-ai/multica/server/internal/issueguard"
	spaceapp "github.com/multica-ai/multica/server/modules/space/application"
	"github.com/multica-ai/multica/server/modules/space/domain"
	spacehttp "github.com/multica-ai/multica/server/modules/space/interfaces/http"
)

// Uploader maps the composed Issue/Space workflow to the installed HTTP contract.
type Uploader struct {
	workflow *issueguard.UploadWorkflow
}

// New creates the cross-context HTTP integration adapter.
func New(workflow *issueguard.UploadWorkflow) *Uploader {
	return &Uploader{workflow: workflow}
}

// Available reports whether the composed workflow can upload.
func (a *Uploader) Available() bool {
	return a != nil && a.workflow != nil && a.workflow.Available()
}

// Upload maps transport input, output and stable error contracts.
func (a *Uploader) Upload(ctx context.Context, request spacehttp.UploadRequest) (spacehttp.UploadResult, error) {
	result, err := a.workflow.Upload(ctx, issueguard.UploadCommand{
		UserID:      request.UserID,
		WorkspaceID: request.WorkspaceID,
		IssueID:     request.IssueID,
		Filename:    request.Filename,
		ContentType: request.ContentType,
		Content:     request.Content,
	})
	err = mapWorkflowError(err)
	asset, assetErr := domainAsset(result.Asset)
	if assetErr != nil {
		return spacehttp.UploadResult{}, assetErr
	}
	return spacehttp.UploadResult{
		Asset:    asset,
		IssueID:  result.IssueID,
		ID:       result.ID,
		URL:      result.URL,
		Filename: result.Filename,
	}, err
}

func mapWorkflowError(err error) error {
	var metadataErr *issueguard.MetadataPersistenceError
	if errors.As(err, &metadataErr) {
		return &spaceapp.MetadataPersistenceError{
			Result: spaceapp.UploadResult{
				ID:       metadataErr.Result.ID,
				URL:      metadataErr.Result.URL,
				Filename: metadataErr.Result.Filename,
			},
			Err: metadataErr.Err,
		}
	}
	for _, mapping := range []errorMapping{
		{issueguard.ErrIssueNotAccessible, spacehttp.ErrIssueNotAccessible},
		{issueguard.ErrInvalidIssueID, spacehttp.ErrInvalidIssueID},
		{issueguard.ErrStorageUnavailable, spaceapp.ErrStorageUnavailable},
		{issueguard.ErrNotWorkspaceMember, spaceapp.ErrNotWorkspaceMember},
		{issueguard.ErrUploadFailed, spaceapp.ErrUploadFailed},
		{issueguard.ErrGenerateID, spaceapp.ErrGenerateID},
	} {
		if errors.Is(err, mapping.from) {
			return errors.Join(mapping.to, err)
		}
	}
	return err
}

type errorMapping struct {
	from error
	to   error
}

func domainAsset(reference *issueguard.PreparedAsset) (*domain.Asset, error) {
	if reference == nil {
		return nil, nil
	}
	asset, err := domain.NewUploadedAsset(domain.UploadedAssetParams{
		ID:           reference.ID,
		WorkspaceID:  reference.WorkspaceID,
		UploaderType: domain.UploaderType(reference.UploaderType),
		UploaderID:   reference.UploaderID,
		Filename:     reference.Filename,
		URL:          reference.URL,
		ContentType:  reference.ContentType,
		SizeBytes:    reference.SizeBytes,
		Checksum:     reference.Checksum,
		CreatedAt:    reference.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	return &asset, nil
}
