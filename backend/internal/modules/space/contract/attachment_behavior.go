package contract

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
)

var (
	ErrAttachmentNotFound       = errors.New("attachment not found")
	ErrAttachmentInvalid        = errors.New("invalid attachment request")
	ErrAttachmentTooLarge       = errors.New("attachment too large")
	ErrAttachmentUnsupported    = errors.New("attachment type is not supported")
	ErrAttachmentTargetNotFound = errors.New("attachment target not found")
	ErrAttachmentForbidden      = errors.New("attachment operation is forbidden")
)

// Attachment is the installed workspace attachment response contract.
type Attachment struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	IssueID       *string `json:"issue_id"`
	CommentID     *string `json:"comment_id"`
	ChatSessionID *string `json:"chat_session_id"`
	ChatMessageID *string `json:"chat_message_id"`
	UploaderType  string  `json:"uploader_type"`
	UploaderID    string  `json:"uploader_id"`
	Filename      string  `json:"filename"`
	URL           string  `json:"url"`
	DownloadURL   string  `json:"download_url"`
	MarkdownURL   string  `json:"markdown_url"`
	ContentType   string  `json:"content_type"`
	SizeBytes     int64   `json:"size_bytes"`
	CreatedAt     string  `json:"created_at"`
}

type AttachmentBinding struct {
	IssueID   *string
	CommentID *string
}

type UploadAttachmentRequest struct {
	WorkspaceID  string
	UploaderType string
	UploaderID   string
	Filename     string
	ContentType  string
	Content      []byte
	IssueID      *string
	CommentID    *string
}

// AttachmentExecutor is the deliberately narrow transaction surface exposed
// to the Workspace-owned relation adapter. It cannot open or commit a
// transaction and therefore cannot take ownership of Space persistence.
type AttachmentExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// AttachmentRelations is implemented by Workspace. Space invokes it while
// holding its own SQLite transaction, keeping relation writes atomic with the
// Asset metadata commit without letting Workspace write Space tables.
type AttachmentRelations interface {
	ResolveBinding(context.Context, string, *string, *string) (AttachmentBinding, error)
	Bind(context.Context, AttachmentExecutor, string, string, AttachmentBinding) error
	Unbind(context.Context, AttachmentExecutor, string, string) error
	Locate(context.Context, string, string) (AttachmentBinding, error)
	ListIssueAssetIDs(context.Context, string, string) ([]string, error)
}

// AttachmentCleanup is a prepared file/metadata deletion participating in a
// caller-owned product database transaction. Commit removes quarantined bytes;
// Rollback restores them. Space remains the only owner of its SQL and paths.
type AttachmentCleanup interface {
	Commit(context.Context)
	Rollback(context.Context) error
}

type AttachmentCleanupService interface {
	PrepareDelete(context.Context, AttachmentExecutor, string, []string) (AttachmentCleanup, error)
}

// AttachmentReferenceValidator validates a complete attachment reference set
// on the caller-owned transaction connection. This keeps validation ordered
// with the Workspace write without transferring Space table ownership.
type AttachmentReferenceValidator interface {
	ValidateReferences(context.Context, AttachmentExecutor, string, []string) error
}

type AttachmentService interface {
	AttachmentCleanupService
	AttachmentReferenceValidator
	Upload(context.Context, UploadAttachmentRequest) (Attachment, error)
	Get(context.Context, string) (Attachment, error)
	ListIssue(context.Context, string, string) ([]Attachment, error)
	Open(context.Context, string) (Attachment, io.ReadCloser, error)
	Delete(context.Context, string) error
	AssetBelongsToWorkspace(context.Context, string, string) (bool, error)
	Reconcile(context.Context) error
}

type HTTPIdentity struct{ WorkspaceID, ActorType, ActorID string }
type HTTPIdentityResolver func(*http.Request) (HTTPIdentity, error)
type HTTPUserResolver func(*http.Request) (string, error)
type HTTPMutationAuthorizer func(*http.Request) error
type WorkspaceMembershipReader func(context.Context, string, string) (role string, found bool, err error)

type WorkspaceEventPublisher interface {
	Publish(workspaceID, eventType string, payload any, actorID, actorType string)
}

type attachmentActorKey struct{}

type AttachmentActor struct{ Type, ID string }

func WithAttachmentActor(ctx context.Context, actorType, actorID string) context.Context {
	return context.WithValue(ctx, attachmentActorKey{}, AttachmentActor{Type: actorType, ID: actorID})
}

func AttachmentActorFromContext(ctx context.Context) (AttachmentActor, bool) {
	value, ok := ctx.Value(attachmentActorKey{}).(AttachmentActor)
	return value, ok
}
