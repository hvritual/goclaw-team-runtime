package contract

import "errors"

var (
	ErrInvalidProject               = errors.New("invalid project")
	ErrProjectNotFound              = errors.New("project not found")
	ErrInvalidProjectActorRelation  = errors.New("invalid project actor relation")
	ErrActorOutsideWorkspace        = errors.New("actor does not belong to workspace")
	ErrInvalidTodo                  = errors.New("invalid todo")
	ErrTodoNotFound                 = errors.New("todo not found")
	ErrWorkspaceActorRequired       = errors.New("workspace actor is required")
	ErrWorkspacePermissionDenied    = errors.New("insufficient workspace role")
	ErrInvalidIssue                 = errors.New("invalid issue")
	ErrIssueNotFound                = errors.New("issue not found")
	ErrInvalidIssueMetadata         = errors.New("invalid issue metadata")
	ErrInvalidKnowledge             = errors.New("invalid knowledge")
	ErrKnowledgeNotFound            = errors.New("knowledge not found")
	ErrAssetOutsideWorkspace        = errors.New("asset does not belong to workspace")
	ErrInvalidRequirement           = errors.New("invalid requirement")
	ErrRequirementNotFound          = errors.New("requirement not found")
	ErrInvalidWorkspaceSetting      = errors.New("invalid workspace setting")
	ErrInvalidWorkspaceSkillBinding = errors.New("invalid workspace skill binding")
	ErrSkillReferenceNotFound       = errors.New("skill reference not found")
)
