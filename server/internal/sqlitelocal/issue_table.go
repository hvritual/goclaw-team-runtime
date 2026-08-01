package sqlitelocal

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const (
	issueTableDefaultPageSize = 50
	issueTableMaxPageSize     = 100
)

type sqliteIssueTableActorRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type sqliteIssueTableScope struct {
	Kind          string                    `json:"kind"`
	AssigneeTypes []string                  `json:"assignee_types,omitempty"`
	ProjectID     string                    `json:"project_id,omitempty"`
	Actor         *sqliteIssueTableActorRef `json:"actor,omitempty"`
	Relation      string                    `json:"relation,omitempty"`
}

type sqliteIssueTableDateFilter struct {
	Field string `json:"field"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type sqliteIssueTableFilters struct {
	Statuses          []string                    `json:"statuses,omitempty"`
	Priorities        []string                    `json:"priorities,omitempty"`
	Assignees         []sqliteIssueTableActorRef  `json:"assignees,omitempty"`
	IncludeNoAssignee bool                        `json:"include_no_assignee,omitempty"`
	Creators          []sqliteIssueTableActorRef  `json:"creators,omitempty"`
	ProjectIDs        []string                    `json:"project_ids,omitempty"`
	IncludeNoProject  bool                        `json:"include_no_project,omitempty"`
	LabelIDs          []string                    `json:"label_ids,omitempty"`
	Properties        map[string][]string         `json:"properties,omitempty"`
	Date              *sqliteIssueTableDateFilter `json:"date,omitempty"`
	IncludeSubIssues  *bool                       `json:"include_sub_issues,omitempty"`
}

type sqliteIssueTableSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type sqliteIssueTableQuery struct {
	Scope   sqliteIssueTableScope   `json:"scope"`
	Filters sqliteIssueTableFilters `json:"filters"`
	Search  string                  `json:"search,omitempty"`
	Sort    sqliteIssueTableSort    `json:"sort"`
}

type sqliteIssueTableGroup struct {
	Kind string `json:"kind"`
}

type sqliteIssueTablePage struct {
	Limit  int     `json:"limit,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

type sqliteIssueTableRowsRequest struct {
	Query     sqliteIssueTableQuery `json:"query"`
	Group     sqliteIssueTableGroup `json:"group"`
	GroupKey  *string               `json:"group_key"`
	Hierarchy struct {
		Enabled bool `json:"enabled"`
	} `json:"hierarchy"`
	ParentID *string              `json:"parent_id"`
	Page     sqliteIssueTablePage `json:"page"`
}

type sqliteIssueTableFacetSpec struct {
	Kind       string `json:"kind"`
	PropertyID string `json:"property_id,omitempty"`
}

type sqliteIssueTableFacetsRequest struct {
	Query        sqliteIssueTableQuery       `json:"query"`
	Facets       []sqliteIssueTableFacetSpec `json:"facets"`
	IncludeTotal *bool                       `json:"include_total,omitempty"`
}

type sqliteIssueTableCursor struct {
	Version     int     `json:"v"`
	Fingerprint string  `json:"query"`
	GroupKey    *string `json:"group_key,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
	Offset      int     `json:"offset"`
}

func issueTableFingerprint(workspaceID string, query sqliteIssueTableQuery) (string, error) {
	encoded, err := json.Marshal(struct {
		WorkspaceID string                `json:"workspace_id"`
		Query       sqliteIssueTableQuery `json:"query"`
	}{WorkspaceID: workspaceID, Query: query})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func issueTablePage(w http.ResponseWriter, page sqliteIssueTablePage, fingerprint string, groupKey, parentID *string) (int, int, bool) {
	limit := page.Limit
	if limit == 0 {
		limit = issueTableDefaultPageSize
	}
	if limit < 1 || limit > issueTableMaxPageSize {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("page.limit must be between 1 and %d", issueTableMaxPageSize))
		return 0, 0, false
	}
	if page.Cursor == nil || strings.TrimSpace(*page.Cursor) == "" {
		return limit, 0, true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(*page.Cursor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cursor")
		return 0, 0, false
	}
	var cursor sqliteIssueTableCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.Version != 1 || cursor.Offset < 0 {
		writeError(w, http.StatusBadRequest, "invalid cursor")
		return 0, 0, false
	}
	if cursor.Fingerprint != fingerprint || !sameOptionalString(cursor.GroupKey, groupKey) || !sameOptionalString(cursor.ParentID, parentID) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "cursor_query_mismatch", "message": "cursor does not belong to this table query",
		})
		return 0, 0, false
	}
	return limit, cursor.Offset, true
}

func issueTableNextCursor(fingerprint string, groupKey, parentID *string, offset int) *string {
	encoded, err := json.Marshal(sqliteIssueTableCursor{
		Version: 1, Fingerprint: fingerprint, GroupKey: groupKey, ParentID: parentID, Offset: offset,
	})
	if err != nil {
		return nil
	}
	value := base64.RawURLEncoding.EncodeToString(encoded)
	return &value
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func actorMatches(actorType, actorID string, values []sqliteIssueTableActorRef) bool {
	for _, actor := range values {
		if actor.Type == actorType && actor.ID == actorID {
			return true
		}
	}
	return false
}

func issuePropertyMatches(rawProperties string, filters map[string][]string) bool {
	if len(filters) == 0 {
		return true
	}
	properties, _ := mapJSON(rawProperties, map[string]any{}).(map[string]any)
	for propertyID, accepted := range filters {
		value, exists := properties[propertyID]
		if !exists {
			return false
		}
		actual := fmt.Sprint(value)
		matched := false
		for _, candidate := range accepted {
			if candidate == actual {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func issueMatchesTableQuery(value issue, query sqliteIssueTableQuery, currentUserID, omittedFacet string) bool {
	switch query.Scope.Kind {
	case "workspace":
		if len(query.Scope.AssigneeTypes) > 0 && (!value.AssigneeType.Valid || !containsString(query.Scope.AssigneeTypes, value.AssigneeType.String)) {
			return false
		}
	case "project":
		if !value.ProjectID.Valid || value.ProjectID.String != query.Scope.ProjectID {
			return false
		}
	case "assignee":
		if query.Scope.Actor == nil || !value.AssigneeType.Valid || !value.AssigneeID.Valid ||
			value.AssigneeType.String != query.Scope.Actor.Type || value.AssigneeID.String != query.Scope.Actor.ID {
			return false
		}
	case "creator":
		if query.Scope.Actor == nil || value.CreatorType != query.Scope.Actor.Type || value.CreatorID != query.Scope.Actor.ID {
			return false
		}
	case "my":
		assigned := value.AssigneeType.String == "member" && value.AssigneeID.String == currentUserID
		created := value.CreatorType == "member" && value.CreatorID == currentUserID
		switch query.Scope.Relation {
		case "assigned":
			if !assigned {
				return false
			}
		case "created":
			if !created {
				return false
			}
		case "", "any":
			if !assigned && !created {
				return false
			}
		default:
			return false
		}
	default:
		return false
	}

	filters := query.Filters
	if omittedFacet != "status" && len(filters.Statuses) > 0 && !containsString(filters.Statuses, value.Status) {
		return false
	}
	if omittedFacet != "priority" && len(filters.Priorities) > 0 && !containsString(filters.Priorities, value.Priority) {
		return false
	}
	if omittedFacet != "assignee" && (filters.Assignees != nil || filters.IncludeNoAssignee) {
		matched := value.AssigneeType.Valid && value.AssigneeID.Valid && actorMatches(value.AssigneeType.String, value.AssigneeID.String, filters.Assignees)
		matched = matched || (filters.IncludeNoAssignee && !value.AssigneeType.Valid && !value.AssigneeID.Valid)
		if !matched {
			return false
		}
	}
	if omittedFacet != "creator" && len(filters.Creators) > 0 && !actorMatches(value.CreatorType, value.CreatorID, filters.Creators) {
		return false
	}
	if omittedFacet != "project" && (len(filters.ProjectIDs) > 0 || filters.IncludeNoProject) {
		matched := value.ProjectID.Valid && containsString(filters.ProjectIDs, value.ProjectID.String)
		matched = matched || (filters.IncludeNoProject && !value.ProjectID.Valid)
		if !matched {
			return false
		}
	}
	// SQLite local mode does not expose labels yet, so a requested label filter
	// intentionally matches no issues instead of silently widening the query.
	if omittedFacet != "label" && len(filters.LabelIDs) > 0 {
		return false
	}
	if !strings.HasPrefix(omittedFacet, "property:") && !issuePropertyMatches(value.Properties, filters.Properties) {
		return false
	}
	if strings.HasPrefix(omittedFacet, "property:") {
		propertyFilters := make(map[string][]string, len(filters.Properties))
		for key, accepted := range filters.Properties {
			if "property:"+key != omittedFacet {
				propertyFilters[key] = accepted
			}
		}
		if !issuePropertyMatches(value.Properties, propertyFilters) {
			return false
		}
	}
	if filters.Date != nil {
		actual := value.CreatedAt
		if filters.Date.Field == "updated_at" {
			actual = value.UpdatedAt
		}
		if filters.Date.Field != "created_at" && filters.Date.Field != "updated_at" {
			return false
		}
		if actual < filters.Date.Start || actual >= filters.Date.End {
			return false
		}
	}
	if filters.IncludeSubIssues != nil && !*filters.IncludeSubIssues && value.ParentIssueID.Valid {
		return false
	}
	search := strings.ToLower(strings.TrimSpace(query.Search))
	if search != "" && !strings.Contains(strings.ToLower(value.Title), search) &&
		(!value.Description.Valid || !strings.Contains(strings.ToLower(value.Description.String), search)) {
		return false
	}
	return true
}

func (s *Server) loadIssueTableValues(r *http.Request, workspaceID string, query sqliteIssueTableQuery, omittedFacet string) ([]issue, bool) {
	rows, err := s.db.QueryContext(r.Context(), `SELECT `+issueColumns()+` FROM issues WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	values := make([]issue, 0)
	for rows.Next() {
		value, err := scanIssue(rows)
		if err != nil {
			return nil, false
		}
		if issueMatchesTableQuery(value, query, currentUserID(r), omittedFacet) {
			values = append(values, value)
		}
	}
	return values, rows.Err() == nil
}

func compareIssueTableValues(left, right issue, order sqliteIssueTableSort) bool {
	field := order.Field
	if field == "" {
		field = "position"
	}
	direction := strings.ToLower(order.Direction)
	if direction == "" {
		direction = "asc"
	}
	comparison := 0
	switch field {
	case "position":
		if left.Position < right.Position {
			comparison = -1
		} else if left.Position > right.Position {
			comparison = 1
		}
	case "title":
		comparison = strings.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title))
	case "status":
		comparison = tableRank(left.Status, []string{"backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled"}) -
			tableRank(right.Status, []string{"backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled"})
	case "priority":
		comparison = tableRank(left.Priority, []string{"urgent", "high", "medium", "low", "none"}) -
			tableRank(right.Priority, []string{"urgent", "high", "medium", "low", "none"})
	case "created_at":
		comparison = strings.Compare(left.CreatedAt, right.CreatedAt)
	case "updated_at":
		comparison = strings.Compare(left.UpdatedAt, right.UpdatedAt)
	case "start_date":
		comparison = compareNullableString(left.StartDate.Valid, left.StartDate.String, right.StartDate.Valid, right.StartDate.String)
	case "due_date":
		comparison = compareNullableString(left.DueDate.Valid, left.DueDate.String, right.DueDate.Valid, right.DueDate.String)
	default:
		if strings.HasPrefix(field, "property:") {
			propertyID := strings.TrimPrefix(field, "property:")
			leftProperties, _ := mapJSON(left.Properties, map[string]any{}).(map[string]any)
			rightProperties, _ := mapJSON(right.Properties, map[string]any{}).(map[string]any)
			comparison = strings.Compare(fmt.Sprint(leftProperties[propertyID]), fmt.Sprint(rightProperties[propertyID]))
		}
	}
	if comparison == 0 {
		comparison = strings.Compare(right.CreatedAt, left.CreatedAt)
		if comparison == 0 {
			comparison = strings.Compare(right.ID, left.ID)
		}
	}
	if field != "position" && direction == "desc" {
		return comparison > 0
	}
	return comparison < 0
}

func tableRank(value string, order []string) int {
	for index, candidate := range order {
		if value == candidate {
			return index
		}
	}
	return len(order)
}

func compareNullableString(leftValid bool, left string, rightValid bool, right string) int {
	if leftValid && !rightValid {
		return -1
	}
	if !leftValid && rightValid {
		return 1
	}
	return strings.Compare(left, right)
}

func issueMatchesTableGroup(value issue, group sqliteIssueTableGroup, groupKey *string) bool {
	if group.Kind == "none" {
		return groupKey == nil || strings.TrimSpace(*groupKey) == ""
	}
	if groupKey == nil {
		return false
	}
	key := strings.TrimSpace(*groupKey)
	switch group.Kind {
	case "status":
		return strings.HasPrefix(key, "status:") && value.Status == strings.TrimPrefix(key, "status:")
	case "assignee":
		if key == "assignee:unassigned" {
			return !value.AssigneeType.Valid && !value.AssigneeID.Valid
		}
		return value.AssigneeType.Valid && value.AssigneeID.Valid && key == "assignee:"+value.AssigneeType.String+":"+value.AssigneeID.String
	case "project":
		if key == "project:none" {
			return !value.ProjectID.Valid
		}
		return value.ProjectID.Valid && key == "project:"+value.ProjectID.String
	case "parent":
		if key == "parent:none" {
			return !value.ParentIssueID.Valid
		}
		return value.ParentIssueID.Valid && key == "parent:"+value.ParentIssueID.String
	default:
		return false
	}
}

func (s *Server) listIssueTableRows(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var request sqliteIssueTableRowsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if !request.Hierarchy.Enabled && request.ParentID != nil {
		writeError(w, http.StatusBadRequest, "parent_id requires hierarchy.enabled=true")
		return
	}
	fingerprint, err := issueTableFingerprint(workspaceValue.ID, request.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to canonicalize table query")
		return
	}
	limit, offset, ok := issueTablePage(w, request.Page, fingerprint, request.GroupKey, request.ParentID)
	if !ok {
		return
	}
	values, ok := s.loadIssueTableValues(r, workspaceValue.ID, request.Query, "")
	if !ok {
		writeError(w, http.StatusInternalServerError, "failed to list table rows")
		return
	}
	membership := make(map[string]issue, len(values))
	for _, value := range values {
		membership[value.ID] = value
	}
	branch := make([]issue, 0, len(values))
	for _, value := range values {
		if !issueMatchesTableGroup(value, request.Group, request.GroupKey) {
			continue
		}
		if request.Hierarchy.Enabled {
			if request.ParentID == nil {
				if value.ParentIssueID.Valid {
					if _, parentVisible := membership[value.ParentIssueID.String]; parentVisible {
						continue
					}
				}
			} else if !value.ParentIssueID.Valid || value.ParentIssueID.String != *request.ParentID {
				continue
			}
		}
		branch = append(branch, value)
	}
	sort.SliceStable(branch, func(left, right int) bool {
		return compareIssueTableValues(branch[left], branch[right], request.Query.Sort)
	})
	if offset > len(branch) {
		offset = len(branch)
	}
	end := offset + limit
	if end > len(branch) {
		end = len(branch)
	}
	page := branch[offset:end]
	responseRows := make([]map[string]any, 0, len(page))
	for _, value := range page {
		childCount := 0
		if request.Hierarchy.Enabled {
			for _, candidate := range values {
				if candidate.ParentIssueID.Valid && candidate.ParentIssueID.String == value.ID && issueMatchesTableGroup(candidate, request.Group, request.GroupKey) {
					childCount++
				}
			}
		}
		responseRows = append(responseRows, map[string]any{
			"issue": value.response(workspaceValue.IssuePrefix), "direct_child_count": childCount,
		})
	}
	var nextCursor *string
	if end < len(branch) {
		nextCursor = issueTableNextCursor(fingerprint, request.GroupKey, request.ParentID, end)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query_fingerprint": fingerprint,
		"group_key":         request.GroupKey,
		"parent_id":         request.ParentID,
		"total":             len(values),
		"rows":              responseRows,
		"branch_total":      len(responseRows),
		"next_cursor":       nextCursor,
	})
}

func facetKey(value issue, facet sqliteIssueTableFacetSpec) (string, bool) {
	switch facet.Kind {
	case "status":
		return value.Status, true
	case "priority":
		return value.Priority, true
	case "assignee":
		if !value.AssigneeType.Valid || !value.AssigneeID.Valid {
			return "__none__", true
		}
		return value.AssigneeType.String + ":" + value.AssigneeID.String, true
	case "creator":
		return value.CreatorType + ":" + value.CreatorID, true
	case "project":
		if !value.ProjectID.Valid {
			return "__none__", true
		}
		return value.ProjectID.String, true
	case "property":
		properties, _ := mapJSON(value.Properties, map[string]any{}).(map[string]any)
		propertyValue, exists := properties[facet.PropertyID]
		if !exists {
			return "__none__", true
		}
		return fmt.Sprint(propertyValue), true
	case "label":
		return "", false
	default:
		return "", false
	}
}

func (s *Server) listIssueTableFacets(w http.ResponseWriter, r *http.Request) {
	workspaceValue, ok := s.resolveWorkspace(w, r)
	if !ok {
		return
	}
	var request sqliteIssueTableFacetsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.Facets) > 32 {
		writeError(w, http.StatusBadRequest, "too many facets")
		return
	}
	fingerprint, err := issueTableFingerprint(workspaceValue.ID, request.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to canonicalize table query")
		return
	}
	allValues, ok := s.loadIssueTableValues(r, workspaceValue.ID, request.Query, "")
	if !ok {
		writeError(w, http.StatusInternalServerError, "failed to list table facets")
		return
	}
	responses := make([]map[string]any, 0, len(request.Facets))
	for _, facet := range request.Facets {
		omitted := facet.Kind
		if facet.Kind == "property" {
			omitted = "property:" + facet.PropertyID
		}
		values, loaded := s.loadIssueTableValues(r, workspaceValue.ID, request.Query, omitted)
		if !loaded {
			writeError(w, http.StatusInternalServerError, "failed to list table facets")
			return
		}
		counts := make(map[string]int)
		for _, value := range values {
			if key, present := facetKey(value, facet); present {
				counts[key]++
			}
		}
		keys := make([]string, 0, len(counts))
		for key := range counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		facetValues := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			facetValues = append(facetValues, map[string]any{"key": key, "count": counts[key]})
		}
		response := map[string]any{"kind": facet.Kind, "values": facetValues}
		if facet.Kind == "property" {
			response["property_id"] = facet.PropertyID
		}
		responses = append(responses, response)
	}
	total := 0
	if request.IncludeTotal == nil || *request.IncludeTotal {
		total = len(allValues)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query_fingerprint": fingerprint, "total": total, "facets": responses,
	})
}
