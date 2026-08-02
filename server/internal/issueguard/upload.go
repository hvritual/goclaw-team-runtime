package issueguard

import (
	"context"
	"errors"
	"time"

	"github.com/multica-ai/multica/server/internal/util"
)

var (
	// ErrIssueNotAccessible reports that an Issue is outside the requested workspace.
	ErrIssueNotAccessible = errors.New("issue not accessible in workspace")
	// ErrInvalidIssueID reports malformed input after workspace authorization.
	ErrInvalidIssueID = errors.New("invalid issue id")
	// ErrNotWorkspaceMember reports a workspace authorization failure.
	ErrNotWorkspaceMember = errors.New("not a member of this workspace")
	// ErrStorageUnavailable reports that the Space provider cannot upload.
	ErrStorageUnavailable = errors.New("file upload not configured")
	// ErrUploadFailed reports a failure to write object bytes.
	ErrUploadFailed = errors.New("upload failed")
	// ErrGenerateID reports a failure to allocate an upload identity.
	ErrGenerateID = errors.New("generate asset id")
)

// SpaceUploadCommand is the consumer-owned contract sent to the Space provider.
type SpaceUploadCommand struct {
	UserID      string
	WorkspaceID string
	Filename    string
	ContentType string
	Content     []byte
}

// PreparedAsset is the consumer-owned Asset reference used by the Issue workflow.
type PreparedAsset struct {
	ID           string
	WorkspaceID  string
	UploaderType string
	UploaderID   string
	Filename     string
	URL          string
	ContentType  string
	SizeBytes    int64
	Checksum     string
	CreatedAt    time.Time
}

// SpaceUploadResult describes an ordinary Space upload through the consumer port.
type SpaceUploadResult struct {
	Asset    *PreparedAsset
	ID       string
	URL      string
	Filename string
}

// SpaceUploads is the Issue workflow's consumer-owned provider contract.
type SpaceUploads interface {
	Available() bool
	Upload(ctx context.Context, command SpaceUploadCommand) (SpaceUploadResult, error)
	PrepareWorkspaceAsset(ctx context.Context, command SpaceUploadCommand) (PreparedAsset, error)
}

// WorkspaceAccess authorizes the actor before any Issue lookup occurs.
type WorkspaceAccess interface {
	IsMember(ctx context.Context, userID, workspaceID string) bool
}

// IssueAttachments validates and atomically persists an Issue attachment.
type IssueAttachments interface {
	ExistsInWorkspace(ctx context.Context, issueID, workspaceID string) bool
	CreateForIssue(ctx context.Context, asset PreparedAsset, issueID string) (PreparedAsset, error)
}

// UploadCommand contains an upload request and its optional consumer relation.
type UploadCommand struct {
	UserID      string
	WorkspaceID string
	IssueID     *string
	Filename    string
	ContentType string
	Content     []byte
}

// UploadResult returns an Asset reference together with the Issue relation rendered by clients.
type UploadResult struct {
	Asset    *PreparedAsset
	IssueID  *string
	ID       string
	URL      string
	Filename string
}

// MetadataPersistenceError carries the compatibility fallback after object upload.
type MetadataPersistenceError struct {
	Result UploadResult
	Err    error
}

func (e *MetadataPersistenceError) Error() string { return "persist asset metadata: " + e.Err.Error() }
func (e *MetadataPersistenceError) Unwrap() error { return e.Err }

// UploadWorkflow coordinates consumer authorization and atomic relation persistence.
type UploadWorkflow struct {
	space      SpaceUploads
	workspaces WorkspaceAccess
	issues     IssueAttachments
}

// NewUploadWorkflow composes Issue attachment upload policies.
func NewUploadWorkflow(space SpaceUploads, workspaces WorkspaceAccess, issues IssueAttachments) *UploadWorkflow {
	return &UploadWorkflow{space: space, workspaces: workspaces, issues: issues}
}

// Available reports whether the underlying Space object store can upload.
func (w *UploadWorkflow) Available() bool {
	return w != nil && w.space != nil && w.space.Available()
}

// Upload delegates ordinary uploads and owns the Issue attachment workflow.
func (w *UploadWorkflow) Upload(ctx context.Context, command UploadCommand) (UploadResult, error) {
	spaceCommand := SpaceUploadCommand{
		UserID:      command.UserID,
		WorkspaceID: command.WorkspaceID,
		Filename:    command.Filename,
		ContentType: command.ContentType,
		Content:     command.Content,
	}
	if command.IssueID == nil || command.WorkspaceID == "" {
		result, err := w.space.Upload(ctx, spaceCommand)
		return uploadResult(result), err
	}

	// Membership is deliberately checked before the Issue lookup so callers
	// cannot infer whether an Issue exists in a workspace they cannot access.
	if w.workspaces == nil || !w.workspaces.IsMember(ctx, command.UserID, command.WorkspaceID) {
		return UploadResult{}, ErrNotWorkspaceMember
	}
	if _, err := util.ParseUUID(*command.IssueID); err != nil {
		return UploadResult{}, ErrInvalidIssueID
	}
	if w.issues == nil || !w.issues.ExistsInWorkspace(ctx, *command.IssueID, command.WorkspaceID) {
		return UploadResult{}, ErrIssueNotAccessible
	}

	asset, err := w.space.PrepareWorkspaceAsset(ctx, spaceCommand)
	if err != nil {
		return UploadResult{}, err
	}
	stored, err := w.issues.CreateForIssue(ctx, asset, *command.IssueID)
	if err != nil {
		return UploadResult{}, &MetadataPersistenceError{
			Result: UploadResult{URL: asset.URL, Filename: asset.Filename},
			Err:    err,
		}
	}
	issueID := *command.IssueID
	return UploadResult{Asset: &stored, IssueID: &issueID}, nil
}

func uploadResult(result SpaceUploadResult) UploadResult {
	return UploadResult{
		Asset:    result.Asset,
		ID:       result.ID,
		URL:      result.URL,
		Filename: result.Filename,
	}
}
