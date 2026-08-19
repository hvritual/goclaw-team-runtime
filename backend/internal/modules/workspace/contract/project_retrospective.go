package contract

import (
	"context"
	"errors"
)

var (
	ErrInvalidProjectRetrospective        = errors.New("invalid Project Retrospective")
	ErrProjectRetrospectiveNotFound       = errors.New("Project Retrospective not found")
	ErrProjectRetrospectiveStateConflict  = errors.New("Project Retrospective state conflict")
	ErrProjectRetrospectiveTargetConflict = errors.New("Project Retrospective action target conflict")
)

type ProjectRetrospectiveActionItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

type ProjectRetrospectiveContent struct {
	Summary     string                           `json:"summary"`
	Successes   []string                         `json:"successes"`
	Problems    []string                         `json:"problems"`
	Lessons     []string                         `json:"lessons"`
	ActionItems []ProjectRetrospectiveActionItem `json:"action_items"`
}

type ProjectRetrospectiveParticipant struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

type ProjectRetrospectiveRevision struct {
	Revision     int64                             `json:"revision"`
	Status       string                            `json:"status"`
	Action       string                            `json:"action"`
	Content      ProjectRetrospectiveContent       `json:"content"`
	Participants []ProjectRetrospectiveParticipant `json:"participants"`
	ActorID      string                            `json:"actor_id"`
	CreatedAt    string                            `json:"created_at"`
}

type ProjectRetrospectiveActionLink struct {
	RetrospectiveID string `json:"retrospective_id"`
	ActionItemID    string `json:"action_item_id"`
	SourceRevision  int64  `json:"source_revision"`
	State           string `json:"state"`
	TargetKind      string `json:"target_kind"`
	TargetID        string `json:"target_id,omitempty"`
	CreatedBy       string `json:"created_by"`
	CreatedAt       string `json:"created_at"`
}

type ProjectRetrospectiveAccess struct {
	CanEdit    bool `json:"can_edit"`
	CanPublish bool `json:"can_publish"`
	CanArchive bool `json:"can_archive"`
}

type ProjectRetrospective struct {
	ID                string                           `json:"id"`
	WorkspaceID       string                           `json:"workspace_id"`
	ProjectID         string                           `json:"project_id"`
	Status            string                           `json:"status"`
	CurrentRevision   int64                            `json:"current_revision"`
	PublishedRevision *int64                           `json:"published_revision,omitempty"`
	CreatedBy         string                           `json:"created_by"`
	CreatedAt         string                           `json:"created_at"`
	UpdatedAt         string                           `json:"updated_at"`
	Current           *ProjectRetrospectiveRevision    `json:"current"`
	History           []ProjectRetrospectiveRevision   `json:"history"`
	ActionLinks       []ProjectRetrospectiveActionLink `json:"action_links"`
	Access            ProjectRetrospectiveAccess       `json:"access"`
}

type ProjectRetrospectiveList struct {
	Retrospectives []ProjectRetrospective `json:"retrospectives"`
	NextCursor     string                 `json:"next_cursor,omitempty"`
}

type ProjectRetrospectiveActionItemInput struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

type ProjectRetrospectiveContentInput struct {
	Summary     string                                `json:"summary"`
	Successes   []string                              `json:"successes"`
	Problems    []string                              `json:"problems"`
	Lessons     []string                              `json:"lessons"`
	ActionItems []ProjectRetrospectiveActionItemInput `json:"action_items"`
}

type ProjectRetrospectiveParticipantInput struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

type CreateProjectRetrospectiveRequest struct {
	Content      ProjectRetrospectiveContentInput       `json:"content"`
	Participants []ProjectRetrospectiveParticipantInput `json:"participants"`
}

type UpdateProjectRetrospectiveRequest struct {
	ExpectedRevision int64                                   `json:"expected_revision"`
	Action           string                                  `json:"action"`
	Content          *ProjectRetrospectiveContentInput       `json:"content,omitempty"`
	Participants     *[]ProjectRetrospectiveParticipantInput `json:"participants,omitempty"`
}

type CreateProjectRetrospectiveTargetRequest struct {
	TargetKind *string `json:"target_kind,omitempty"`
}

type ProjectRetrospectiveService interface {
	ListProjectRetrospectives(context.Context, string, string, int, string, bool) (ProjectRetrospectiveList, error)
	GetProjectRetrospective(context.Context, string, string, string) (ProjectRetrospective, error)
	CreateProjectRetrospective(context.Context, string, string, string, CreateProjectRetrospectiveRequest) (ProjectRetrospective, error)
	UpdateProjectRetrospective(context.Context, string, string, string, UpdateProjectRetrospectiveRequest) (ProjectRetrospective, error)
	ArchiveProjectRetrospective(context.Context, string, string, string, int64) (ProjectRetrospective, error)
	CreateProjectRetrospectiveTarget(context.Context, string, string, string, string, string, CreateProjectRetrospectiveTargetRequest) (ProjectRetrospectiveActionLink, error)
}

type IdempotentCreateIssueRequest struct {
	CreateIssueRequest
	IdempotencyKey string
	RequestHash    string
}

type IdempotentCreateIssueResponse struct {
	Issue    *Issue
	Replayed bool
}

type IdempotentIssueCreationService interface {
	CreateIssueIdempotently(context.Context, IdempotentCreateIssueRequest) (IdempotentCreateIssueResponse, error)
}
