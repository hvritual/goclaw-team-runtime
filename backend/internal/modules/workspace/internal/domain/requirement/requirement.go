package requirement

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid requirement")

type Requirement struct {
	ID, WorkspaceID, ProjectID, Title, ApprovalStatus, CoverageStatus string
	CurrentVersion                                                    int32
	IssueIDs                                                          []string
	CreatedAt, UpdatedAt                                              time.Time
}
type Version struct {
	ID, RequirementID string
	Version           int32
	Content           string
	CreatedAt         time.Time
}

func New(id, workspaceID, projectID, title string, issueIDs []string, now time.Time) (Requirement, error) {
	id, workspaceID, projectID, title = strings.TrimSpace(id), strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), strings.TrimSpace(title)
	if id == "" || workspaceID == "" || projectID == "" || title == "" {
		return Requirement{}, ErrInvalid
	}
	issues, err := cleanIDs(issueIDs)
	if err != nil {
		return Requirement{}, err
	}
	coverage := "uncovered"
	if len(issues) > 0 {
		coverage = "covered"
	}
	now = now.UTC()
	return Requirement{ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Title: title, ApprovalStatus: "draft", CoverageStatus: coverage, CurrentVersion: 1, IssueIDs: issues, CreatedAt: now, UpdatedAt: now}, nil
}

func Rehydrate(value Requirement) (Requirement, error) {
	base, err := New(value.ID, value.WorkspaceID, value.ProjectID, value.Title, value.IssueIDs, value.CreatedAt)
	if err != nil {
		return Requirement{}, err
	}
	if value.CurrentVersion < 1 {
		return Requirement{}, ErrInvalid
	}
	base.CurrentVersion, base.ApprovalStatus, base.CoverageStatus, base.UpdatedAt = value.CurrentVersion, value.ApprovalStatus, value.CoverageStatus, value.UpdatedAt.UTC()
	return base, nil
}

func (r Requirement) NextVersion(title string, issueIDs []string, now time.Time) (Requirement, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Requirement{}, ErrInvalid
	}
	issues, err := cleanIDs(issueIDs)
	if err != nil {
		return Requirement{}, err
	}
	r.Title, r.IssueIDs, r.CurrentVersion, r.ApprovalStatus, r.UpdatedAt = title, issues, r.CurrentVersion+1, "draft", now.UTC()
	r.CoverageStatus = "uncovered"
	if len(issues) > 0 {
		r.CoverageStatus = "covered"
	}
	return r, nil
}

func NewVersion(id, requirementID string, version int32, content string, now time.Time) (Version, error) {
	id, requirementID, content = strings.TrimSpace(id), strings.TrimSpace(requirementID), strings.TrimSpace(content)
	if id == "" || requirementID == "" || version < 1 || content == "" {
		return Version{}, ErrInvalid
	}
	return Version{ID: id, RequirementID: requirementID, Version: version, Content: content, CreatedAt: now.UTC()}, nil
}

func cleanIDs(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, ErrInvalid
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}
