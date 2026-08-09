package issue

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StatusBacklog    = "backlog"
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusInReview   = "in_review"
	StatusDone       = "done"
	StatusBlocked    = "blocked"
	StatusCancelled  = "cancelled"

	PriorityUrgent = "urgent"
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
	PriorityNone   = "none"
)

var ErrInvalid = errors.New("invalid issue")

type Issue struct {
	ID, WorkspaceID, Identifier, Title, Status, Priority                                string
	Number                                                                              int32
	Description, AssigneeType, AssigneeID, ParentIssueID, ProjectID, StartDate, DueDate *string
	CreatorType, CreatorID                                                              string
	Position                                                                            float64
	Stage                                                                               *int32
	CreatedAt, UpdatedAt                                                                time.Time
	Metadata, Properties                                                                map[string]any
	AssetIDs                                                                            []string
}

type StringChange struct {
	Set   bool
	Value *string
}

type StageChange struct {
	Set   bool
	Value *int32
}

type AssetsChange struct {
	Set    bool
	Values []string
}

type Patch struct {
	Title, Description, Status, Priority *string
	AssigneeType, AssigneeID             StringChange
	ParentIssueID, ProjectID             StringChange
	Position                             *float64
	Stage                                StageChange
	StartDate, DueDate                   StringChange
	AssetIDs                             AssetsChange
}

func ValidStatus(status string) bool {
	switch status {
	case StatusBacklog, StatusTodo, StatusInProgress, StatusInReview, StatusDone, StatusBlocked, StatusCancelled:
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

func New(id, workspaceID, title string, description *string, status, priority string, assigneeType, assigneeID, parentIssueID, projectID *string, creatorType, creatorID string, position float64, stage *int32, startDate, dueDate *string, assetIDs []string, now time.Time) (Issue, error) {
	status = defaultValue(status, StatusTodo)
	priority = defaultValue(priority, PriorityNone)
	value := Issue{
		ID: strings.TrimSpace(id), WorkspaceID: strings.TrimSpace(workspaceID), Title: strings.TrimSpace(title),
		Description: copyString(description), Status: status, Priority: priority,
		AssigneeType: cleanString(assigneeType), AssigneeID: cleanString(assigneeID),
		ParentIssueID: cleanString(parentIssueID), ProjectID: cleanString(projectID),
		CreatorType: strings.TrimSpace(creatorType), CreatorID: strings.TrimSpace(creatorID), Position: position,
		Stage: copyInt32(stage), StartDate: cleanDate(startDate), DueDate: cleanDate(dueDate),
		Metadata: map[string]any{}, Properties: map[string]any{}, AssetIDs: append([]string{}, assetIDs...),
		CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if err := value.validate(false); err != nil {
		return Issue{}, err
	}
	return value, nil
}

func Rehydrate(value Issue) (Issue, error) {
	value.ID = strings.TrimSpace(value.ID)
	value.WorkspaceID = strings.TrimSpace(value.WorkspaceID)
	value.Identifier = strings.TrimSpace(value.Identifier)
	value.Title = strings.TrimSpace(value.Title)
	value.CreatedAt, value.UpdatedAt = value.CreatedAt.UTC(), value.UpdatedAt.UTC()
	value.Description = copyString(value.Description)
	value.AssigneeType, value.AssigneeID = cleanString(value.AssigneeType), cleanString(value.AssigneeID)
	value.ParentIssueID, value.ProjectID = cleanString(value.ParentIssueID), cleanString(value.ProjectID)
	value.Stage = copyInt32(value.Stage)
	value.StartDate, value.DueDate = cleanDate(value.StartDate), cleanDate(value.DueDate)
	value.Metadata, value.Properties = cloneMap(value.Metadata), cloneMap(value.Properties)
	value.AssetIDs = append([]string{}, value.AssetIDs...)
	if err := value.validate(true); err != nil {
		return Issue{}, err
	}
	return value, nil
}

func (i Issue) AssignIdentity(number int32, prefix string) (Issue, error) {
	prefix = strings.TrimSpace(prefix)
	if number < 1 || prefix == "" {
		return Issue{}, fmt.Errorf("%w: issue number and prefix are required", ErrInvalid)
	}
	i.Number = number
	i.Identifier = fmt.Sprintf("%s-%d", prefix, number)
	if err := i.validate(true); err != nil {
		return Issue{}, err
	}
	return i, nil
}

func (i Issue) Apply(patch Patch, now time.Time) (Issue, error) {
	updated := i
	if patch.Title != nil {
		updated.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		updated.Description = copyString(patch.Description)
	}
	if patch.Status != nil {
		updated.Status = *patch.Status
	}
	if patch.Priority != nil {
		updated.Priority = *patch.Priority
	}
	applyStringChange(&updated.AssigneeType, patch.AssigneeType)
	applyStringChange(&updated.AssigneeID, patch.AssigneeID)
	applyStringChange(&updated.ParentIssueID, patch.ParentIssueID)
	applyStringChange(&updated.ProjectID, patch.ProjectID)
	if patch.Position != nil {
		updated.Position = *patch.Position
	}
	if patch.Stage.Set {
		updated.Stage = copyInt32(patch.Stage.Value)
	}
	applyDateChange(&updated.StartDate, patch.StartDate)
	applyDateChange(&updated.DueDate, patch.DueDate)
	if patch.AssetIDs.Set {
		updated.AssetIDs = append([]string{}, patch.AssetIDs.Values...)
	}
	updated.UpdatedAt = now.UTC()
	if err := updated.validate(true); err != nil {
		return Issue{}, err
	}
	return updated, nil
}

func (i Issue) WithStatus(status string, now time.Time) (Issue, error) {
	return i.Apply(Patch{Status: &status}, now)
}

func (i Issue) validate(numbered bool) error {
	if i.ID == "" || i.WorkspaceID == "" || i.Title == "" || i.CreatorID == "" {
		return fmt.Errorf("%w: identity, workspace, title, and creator are required", ErrInvalid)
	}
	if numbered && (i.Number < 1 || i.Identifier == "") {
		return fmt.Errorf("%w: allocated identifier is required", ErrInvalid)
	}
	if !ValidStatus(i.Status) || !ValidPriority(i.Priority) {
		return fmt.Errorf("%w: invalid status or priority", ErrInvalid)
	}
	if !validActorType(i.CreatorType) || (i.AssigneeType == nil) != (i.AssigneeID == nil) {
		return fmt.Errorf("%w: invalid actor pair", ErrInvalid)
	}
	if i.AssigneeType != nil && !validActorType(*i.AssigneeType) {
		return fmt.Errorf("%w: invalid assignee type", ErrInvalid)
	}
	if i.ParentIssueID != nil && *i.ParentIssueID == i.ID {
		return fmt.Errorf("%w: issue cannot parent itself", ErrInvalid)
	}
	if i.Stage != nil && *i.Stage < 1 {
		return fmt.Errorf("%w: stage must be at least one", ErrInvalid)
	}
	if !validDate(i.StartDate) || !validDate(i.DueDate) {
		return fmt.Errorf("%w: dates must use YYYY-MM-DD", ErrInvalid)
	}
	seen := make(map[string]struct{}, len(i.AssetIDs))
	for _, id := range i.AssetIDs {
		if id != strings.TrimSpace(id) || id == "" {
			return fmt.Errorf("%w: asset id is required", ErrInvalid)
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("%w: duplicate asset id", ErrInvalid)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validActorType(value string) bool { return value == "member" || value == "agent" }

func validDate(value *string) bool {
	if value == nil {
		return true
	}
	parsed, err := time.Parse("2006-01-02", *value)
	return err == nil && parsed.Format("2006-01-02") == *value
}

func defaultValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func cleanString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cleanDate(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return copyString(value)
}

func copyInt32(value *int32) *int32 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func applyStringChange(target **string, change StringChange) {
	if change.Set {
		*target = cleanString(change.Value)
	}
}

func applyDateChange(target **string, change StringChange) {
	if change.Set {
		*target = cleanDate(change.Value)
	}
}

func cloneMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		result := make([]any, len(typed))
		for index := range typed {
			result[index] = cloneValue(typed[index])
		}
		return result
	default:
		return typed
	}
}
