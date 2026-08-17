package contract

import (
	"context"
	"errors"
)

var (
	ErrInvalidTaskPromotion  = errors.New("invalid task promotion")
	ErrTaskAlreadyLinked     = errors.New("task is already linked to an issue")
	ErrTaskPromotionConflict = errors.New("task cannot be promoted in its current state")
)

type PromoteTaskRequest struct {
	WorkspaceId      string `json:"workspaceId,omitempty"`
	TaskId           string `json:"taskId,omitempty"`
	ExpectedRevision int64  `json:"expectedRevision,omitempty"`
	CompleteTask     bool   `json:"completeTask,omitempty"`
	IdempotencyKey   string `json:"-"`
}

type PromoteTaskResponse struct {
	Task         *Todo  `json:"task,omitempty"`
	Issue        *Issue `json:"issue,omitempty"`
	SourceTaskId string `json:"sourceTaskId,omitempty"`
}

type TaskPromotionService interface {
	PromoteTask(context.Context, PromoteTaskRequest) (PromoteTaskResponse, error)
}
