package contract

import "context"

type CommentReaction struct {
	ID        string
	CommentID string
	ActorType string
	ActorID   string
	Emoji     string
	CreatedAt string
}

type IssueReaction struct {
	ID        string
	IssueID   string
	ActorType string
	ActorID   string
	Emoji     string
	CreatedAt string
}

type IssueComment struct {
	ID             string
	WorkspaceID    string
	IssueID        string
	AuthorType     string
	AuthorID       string
	Content        string
	Type           string
	ParentID       *string
	Reactions      []CommentReaction
	Attachments    []map[string]any
	AttachmentIDs  []string
	CreatedAt      string
	UpdatedAt      string
	ResolvedAt     *string
	ResolvedByType *string
	ResolvedByID   *string
}

type IssueAttachmentProjectionReader interface {
	ReadAttachments(context.Context, string, []string) ([]map[string]any, error)
}

type IssueActivity struct {
	ID        string
	IssueID   string
	ActorType string
	ActorID   string
	Action    string
	Details   map[string]any
	CreatedAt string
}

type IssueTimelineEntry struct {
	Type           string
	ID             string
	ActorType      string
	ActorID        string
	CreatedAt      string
	Action         *string
	Details        map[string]any
	Content        *string
	ParentID       *string
	UpdatedAt      *string
	CommentType    *string
	Reactions      []CommentReaction
	Attachments    []map[string]any
	ResolvedAt     *string
	ResolvedByType *string
	ResolvedByID   *string
}

type IssueSubscriber struct {
	IssueID   string
	UserType  string
	UserID    string
	Reason    string
	CreatedAt string
}

type ListIssueCommentsRequest struct{ WorkspaceID, IssueID string }
type ListIssueCommentsResponse struct{ Comments []IssueComment }
type ListIssueTimelineRequest struct{ WorkspaceID, IssueID string }
type ListIssueTimelineResponse struct{ Entries []IssueTimelineEntry }

type CreateIssueCommentRequest struct {
	WorkspaceID   string
	IssueID       string
	Content       string
	Type          string
	ParentID      *string
	AttachmentIDs []string
}

type UpdateIssueCommentRequest struct {
	WorkspaceID   string
	CommentID     string
	Content       string
	AttachmentIDs []string
}

type DeleteIssueCommentRequest struct{ WorkspaceID, CommentID string }
type ResolveIssueCommentRequest struct {
	WorkspaceID, CommentID string
	Resolved               bool
}
type ProposeCommentKnowledgeRequest struct{ WorkspaceID, CommentID string }
type CommentKnowledgeProposalResponse struct {
	Queued         bool
	EvidenceID     *string
	SourceRevision string
}

type ChangeCommentReactionRequest struct{ WorkspaceID, CommentID, Emoji string }
type ChangeIssueReactionRequest struct{ WorkspaceID, IssueID, Emoji string }
type ListIssueReactionsRequest struct{ WorkspaceID, IssueID string }
type ListIssueReactionsResponse struct{ Reactions []IssueReaction }

type ListIssueSubscribersRequest struct{ WorkspaceID, IssueID string }
type ListIssueSubscribersResponse struct{ Subscribers []IssueSubscriber }
type ChangeIssueSubscriberRequest struct{ WorkspaceID, IssueID, UserType, UserID string }

type IssueCollaborationService interface {
	ResolveIssueID(context.Context, string, string) (string, error)
	GetIssueComment(context.Context, string, string) (IssueComment, error)
	ListIssueComments(context.Context, ListIssueCommentsRequest) (ListIssueCommentsResponse, error)
	ListIssueTimeline(context.Context, ListIssueTimelineRequest) (ListIssueTimelineResponse, error)
	CreateIssueComment(context.Context, CreateIssueCommentRequest) (IssueComment, error)
	UpdateIssueComment(context.Context, UpdateIssueCommentRequest) (IssueComment, error)
	DeleteIssueComment(context.Context, DeleteIssueCommentRequest) error
	ResolveIssueComment(context.Context, ResolveIssueCommentRequest) (IssueComment, error)
	ProposeCommentKnowledge(context.Context, ProposeCommentKnowledgeRequest) (CommentKnowledgeProposalResponse, error)
	AddCommentReaction(context.Context, ChangeCommentReactionRequest) (CommentReaction, error)
	RemoveCommentReaction(context.Context, ChangeCommentReactionRequest) error
	ListIssueReactions(context.Context, ListIssueReactionsRequest) (ListIssueReactionsResponse, error)
	AddIssueReaction(context.Context, ChangeIssueReactionRequest) (IssueReaction, error)
	RemoveIssueReaction(context.Context, ChangeIssueReactionRequest) error
	ListIssueSubscribers(context.Context, ListIssueSubscribersRequest) (ListIssueSubscribersResponse, error)
	SubscribeToIssue(context.Context, ChangeIssueSubscriberRequest) error
	UnsubscribeFromIssue(context.Context, ChangeIssueSubscriberRequest) error
}

type IssueActivityRecorder interface {
	RecordIssueActivity(context.Context, string, string, string, map[string]any) (IssueActivity, error)
}
