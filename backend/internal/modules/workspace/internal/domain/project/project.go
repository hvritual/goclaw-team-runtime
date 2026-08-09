package project

import (
	"errors"
	"strings"
	"time"
)

const (
	StatusPlanned    = "planned"
	StatusInProgress = "in_progress"
	StatusPaused     = "paused"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
)

var (
	ErrIDRequired        = errors.New("project id is required")
	ErrWorkspaceRequired = errors.New("workspace id is required")
	ErrNameRequired      = errors.New("project name is required")
	ErrInvalidStatus     = errors.New("invalid project status")
	ErrAssetIDRequired   = errors.New("project asset id is required")
)

var validStatuses = map[string]struct{}{
	StatusPlanned: {}, StatusInProgress: {}, StatusPaused: {},
	StatusCompleted: {}, StatusCancelled: {},
}

func IsValidStatus(status string) bool {
	_, ok := validStatuses[strings.TrimSpace(status)]
	return ok
}

type Project struct {
	id          string
	workspaceID string
	name        string
	description string
	status      string
	assetIDs    []string
	createdAt   time.Time
	updatedAt   time.Time
}

func New(id, workspaceID, name, description, status string, assetIDs []string, now time.Time) (Project, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	name = strings.TrimSpace(name)
	status = strings.TrimSpace(status)
	if id == "" {
		return Project{}, ErrIDRequired
	}
	if workspaceID == "" {
		return Project{}, ErrWorkspaceRequired
	}
	if name == "" {
		return Project{}, ErrNameRequired
	}
	if status == "" {
		status = StatusPlanned
	}
	if _, ok := validStatuses[status]; !ok {
		return Project{}, ErrInvalidStatus
	}
	normalizedAssets := make([]string, len(assetIDs))
	for index, assetID := range assetIDs {
		normalizedAssets[index] = strings.TrimSpace(assetID)
		if normalizedAssets[index] == "" {
			return Project{}, ErrAssetIDRequired
		}
	}
	now = now.UTC()
	return Project{
		id: id, workspaceID: workspaceID, name: name,
		description: description, status: status, assetIDs: normalizedAssets,
		createdAt: now, updatedAt: now,
	}, nil
}

func Rehydrate(id, workspaceID, name, description, status string, assetIDs []string, createdAt, updatedAt time.Time) (Project, error) {
	value, err := New(id, workspaceID, name, description, status, assetIDs, createdAt)
	if err != nil {
		return Project{}, err
	}
	value.updatedAt = updatedAt.UTC()
	return value, nil
}

func (p Project) Update(name, description, status *string, now time.Time) (Project, error) {
	updated := p
	if name != nil {
		updated.name = strings.TrimSpace(*name)
		if updated.name == "" {
			return Project{}, ErrNameRequired
		}
	}
	if description != nil {
		updated.description = *description
	}
	if status != nil {
		updated.status = strings.TrimSpace(*status)
		if !IsValidStatus(updated.status) {
			return Project{}, ErrInvalidStatus
		}
	}
	updated.updatedAt = now.UTC()
	return updated, nil
}

func (p Project) ID() string           { return p.id }
func (p Project) WorkspaceID() string  { return p.workspaceID }
func (p Project) Name() string         { return p.name }
func (p Project) Description() string  { return p.description }
func (p Project) Status() string       { return p.status }
func (p Project) AssetIDs() []string   { return append([]string(nil), p.assetIDs...) }
func (p Project) CreatedAt() time.Time { return p.createdAt }
func (p Project) UpdatedAt() time.Time { return p.updatedAt }
