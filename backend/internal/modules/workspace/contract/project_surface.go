package contract

import (
	"context"
	"encoding/json"
)

type ProjectSurfaceProject struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	Icon          *string `json:"icon"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	LeadType      *string `json:"lead_type"`
	LeadID        *string `json:"lead_id"`
	StartDate     *string `json:"start_date"`
	DueDate       *string `json:"due_date"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	IssueCount    int     `json:"issue_count"`
	DoneCount     int     `json:"done_count"`
	ResourceCount int     `json:"resource_count"`
}

type ProjectSurfaceList struct {
	Projects []ProjectSurfaceProject `json:"projects"`
	Total    int                     `json:"total"`
}

type ProjectSurfaceSearchRequest struct {
	Query         string
	IncludeClosed bool
	Limit         int
	Offset        int
}

type ProjectSurfaceSearchResult struct {
	ProjectSurfaceProject
	MatchSource    string  `json:"match_source"`
	MatchedSnippet *string `json:"matched_snippet,omitempty"`
}

type ProjectSurfaceSearchResponse struct {
	Projects []ProjectSurfaceSearchResult `json:"projects"`
	Total    int                          `json:"total"`
}

type Pin struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	UserID      string  `json:"user_id"`
	ItemType    string  `json:"item_type"`
	ItemID      string  `json:"item_id"`
	Position    float64 `json:"position"`
	CreatedAt   string  `json:"created_at"`
}

type CreateProjectSurfaceRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	LeadType    *string `json:"lead_type"`
	LeadID      *string `json:"lead_id"`
	StartDate   *string `json:"start_date"`
	DueDate     *string `json:"due_date"`
}

type NullableStringPatch struct {
	Set   bool
	Value *string
}

func (p *NullableStringPatch) UnmarshalJSON(data []byte) error {
	p.Set = true
	if string(data) == "null" {
		p.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.Value = &value
	return nil
}

type UpdateProjectSurfaceRequest struct {
	Title       *string             `json:"title"`
	Description NullableStringPatch `json:"description"`
	Icon        NullableStringPatch `json:"icon"`
	Status      *string             `json:"status"`
	Priority    *string             `json:"priority"`
	LeadType    NullableStringPatch `json:"lead_type"`
	LeadID      NullableStringPatch `json:"lead_id"`
	StartDate   NullableStringPatch `json:"start_date"`
	DueDate     NullableStringPatch `json:"due_date"`
}

type CreatePinRequest struct {
	ItemType string `json:"item_type"`
	ItemID   string `json:"item_id"`
}

type ProjectSurfaceService interface {
	ListProjects(context.Context, string, string) (ProjectSurfaceList, error)
	GetProject(context.Context, string, string) (ProjectSurfaceProject, error)
	CreateProject(context.Context, string, CreateProjectSurfaceRequest) (ProjectSurfaceProject, error)
	UpdateProject(context.Context, string, string, UpdateProjectSurfaceRequest) (ProjectSurfaceProject, error)
	DeleteProject(context.Context, string, string) error
	ListPins(context.Context, string, string) ([]Pin, error)
	CreatePin(context.Context, string, string, CreatePinRequest) (Pin, error)
	DeletePin(context.Context, string, string, string, string) error
}

type ProjectSurfaceSearchService interface {
	SearchProjects(context.Context, string, ProjectSurfaceSearchRequest) (ProjectSurfaceSearchResponse, error)
}
