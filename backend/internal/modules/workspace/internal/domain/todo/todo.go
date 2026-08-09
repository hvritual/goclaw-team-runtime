package todo

import (
	"errors"
	"strings"
	"time"
)

const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusCancelled  = "cancelled"

	PriorityUrgent = "urgent"
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
	PriorityNone   = "none"
)

var ErrInvalid = errors.New("invalid todo")

// Todo is an ordinary Workspace task. Agent execution lifecycle is outside
// this aggregate.
type Todo struct {
	ID, WorkspaceID, Title, Description, Status, Priority string
	ProjectID, IssueID, AssigneeType, AssigneeID          *string
	CreatorType, CreatorID                                string
	Position                                              float64
	StartDate, DueDate, CompletedAt                       *time.Time
	CreatedAt, UpdatedAt                                  time.Time
}

// StringChange and TimeChange distinguish an omitted update from an explicit
// clear. A Set change with a nil value clears the field.
type StringChange struct {
	Set   bool
	Value *string
}

type TimeChange struct {
	Set   bool
	Value *time.Time
}

type Patch struct {
	Title, Description, Status, Priority *string
	ProjectID, IssueID                   StringChange
	AssigneeType, AssigneeID             StringChange
	Position                             *float64
	StartDate, DueDate                   TimeChange
}

func New(id, workspaceID, title, description, status, priority string, projectID, issueID, assigneeType, assigneeID *string, creatorType, creatorID string, position float64, startDate, dueDate *time.Time, now time.Time) (Todo, error) {
	value := Todo{
		ID: id, WorkspaceID: workspaceID, Title: title, Description: description,
		Status: status, Priority: priority, ProjectID: copyCleanString(projectID),
		IssueID: copyCleanString(issueID), AssigneeType: copyCleanString(assigneeType),
		AssigneeID: copyCleanString(assigneeID), CreatorType: creatorType,
		CreatorID: creatorID, Position: position, StartDate: copyTime(startDate),
		DueDate: copyTime(dueDate), CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	value.ID = strings.TrimSpace(value.ID)
	value.WorkspaceID = strings.TrimSpace(value.WorkspaceID)
	value.Title = strings.TrimSpace(value.Title)
	value.CreatorType = strings.TrimSpace(value.CreatorType)
	value.CreatorID = strings.TrimSpace(value.CreatorID)
	if value.Status == "" {
		value.Status = StatusTodo
	}
	if value.Priority == "" {
		value.Priority = PriorityNone
	}
	if err := value.validate(true); err != nil {
		return Todo{}, err
	}
	return value, nil
}

func Rehydrate(value Todo) (Todo, error) {
	value.ID = strings.TrimSpace(value.ID)
	value.WorkspaceID = strings.TrimSpace(value.WorkspaceID)
	value.Title = strings.TrimSpace(value.Title)
	value.CreatorType = strings.TrimSpace(value.CreatorType)
	value.CreatorID = strings.TrimSpace(value.CreatorID)
	value.ProjectID = copyCleanString(value.ProjectID)
	value.IssueID = copyCleanString(value.IssueID)
	value.AssigneeType = copyCleanString(value.AssigneeType)
	value.AssigneeID = copyCleanString(value.AssigneeID)
	value.StartDate = copyTime(value.StartDate)
	value.DueDate = copyTime(value.DueDate)
	value.CompletedAt = copyTime(value.CompletedAt)
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	if err := value.validate(true); err != nil {
		return Todo{}, err
	}
	return value, nil
}

func (t Todo) Apply(patch Patch, now time.Time) (Todo, error) {
	next := t
	if patch.Title != nil {
		next.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		next.Description = *patch.Description
	}
	if patch.Status != nil {
		next.Status = *patch.Status
	}
	if patch.Priority != nil {
		next.Priority = *patch.Priority
	}
	if patch.ProjectID.Set {
		next.ProjectID = copyCleanString(patch.ProjectID.Value)
	}
	if patch.IssueID.Set {
		next.IssueID = copyCleanString(patch.IssueID.Value)
	}
	if patch.AssigneeType.Set {
		next.AssigneeType = copyCleanString(patch.AssigneeType.Value)
	}
	if patch.AssigneeID.Set {
		next.AssigneeID = copyCleanString(patch.AssigneeID.Value)
	}
	if patch.AssigneeType.Set && next.AssigneeType == nil || patch.AssigneeID.Set && next.AssigneeID == nil {
		next.AssigneeType, next.AssigneeID = nil, nil
	}
	if patch.Position != nil {
		next.Position = *patch.Position
	}
	if patch.StartDate.Set {
		next.StartDate = copyTime(patch.StartDate.Value)
	}
	if patch.DueDate.Set {
		next.DueDate = copyTime(patch.DueDate.Value)
	}
	if err := next.validate(true); err != nil {
		return Todo{}, err
	}
	if patch.Status != nil {
		if next.Status == StatusDone {
			if t.Status != StatusDone || t.CompletedAt == nil {
				completed := now.UTC()
				next.CompletedAt = &completed
			}
		} else {
			next.CompletedAt = nil
		}
	}
	next.UpdatedAt = now.UTC()
	return next, nil
}

func (t Todo) WithStatus(status string, now time.Time) (Todo, error) {
	return t.Apply(Patch{Status: &status}, now)
}

func (t Todo) validate(requireCreator bool) error {
	if t.ID == "" || t.WorkspaceID == "" || t.Title == "" || !ValidStatus(t.Status) || !ValidPriority(t.Priority) {
		return ErrInvalid
	}
	if (t.AssigneeType == nil) != (t.AssigneeID == nil) {
		return ErrInvalid
	}
	if t.AssigneeType != nil && !validActorType(*t.AssigneeType) {
		return ErrInvalid
	}
	if requireCreator && (!validActorType(t.CreatorType) || t.CreatorID == "") {
		return ErrInvalid
	}
	return nil
}

func ValidStatus(status string) bool {
	switch status {
	case StatusTodo, StatusInProgress, StatusDone, StatusCancelled:
		return true
	default:
		return false
	}
}

func ValidPriority(priority string) bool {
	switch priority {
	case PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone:
		return true
	default:
		return false
	}
}

func validActorType(actorType string) bool {
	return actorType == "member" || actorType == "agent"
}

func copyCleanString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}
