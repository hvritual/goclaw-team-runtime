package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type IssueReadHandler struct {
	service       contract.IssueService
	identity      contract.WorkspaceHTTPIdentityResolver
	authenticate  func(*http.Request) (string, error)
	mutation      func(*http.Request) error
	createEnabled bool
}

var errUnsupportedIssueQuery = errors.New("unsupported issue query")

func NewIssueReadHandler(service contract.IssueService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error, createEnabled bool) *IssueReadHandler {
	return &IssueReadHandler{service: service, identity: identity, authenticate: authenticate, mutation: mutation, createEnabled: createEnabled}
}

func (h *IssueReadHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/api/issues", h.list)
	if h.createEnabled {
		router.POST("/api/issues", h.create)
	}
	router.POST("/api/issues/query", h.query)
	router.POST("/api/issues/table/facets", h.facets)
	router.POST("/api/issues/table/groups", h.groups)
	router.POST("/api/issues/table/rows", h.rows)
	router.GET("/api/issues/{id}", h.get)
}

type createIssueHTTPRequest struct {
	Title         string   `json:"title"`
	Description   *string  `json:"description"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	AssigneeType  *string  `json:"assignee_type"`
	AssigneeID    *string  `json:"assignee_id"`
	ParentIssueID *string  `json:"parent_issue_id"`
	ProjectID     *string  `json:"project_id"`
	Position      float64  `json:"position"`
	Stage         *int32   `json:"stage"`
	StartDate     *string  `json:"start_date"`
	DueDate       *string  `json:"due_date"`
	AttachmentIDs []string `json:"attachment_ids"`
	LabelIDs      []string `json:"label_ids"`
}

func (h *IssueReadHandler) create(ctx kratoshttp.Context) error {
	if h.authenticate == nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if _, err := h.authenticate(ctx.Request()); err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	if !hasWorkspaceIdentity(ctx) {
		return writeError(ctx, http.StatusBadRequest, "workspace is required")
	}
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	if h.mutation == nil || h.mutation(ctx.Request()) != nil {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	var request createIssueHTTPRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	if len(request.AttachmentIDs) != 0 || len(request.LabelIDs) != 0 {
		return writeError(ctx, http.StatusBadRequest, "unsupported issue create field")
	}
	result, err := h.service.CreateIssue(requestContext, contract.CreateIssueRequest{
		WorkspaceId: workspaceID, Title: request.Title, Description: request.Description,
		Status: request.Status, Priority: request.Priority, AssigneeType: request.AssigneeType,
		AssigneeId: request.AssigneeID, ParentIssueId: request.ParentIssueID, ProjectId: request.ProjectID,
		Position: request.Position, Stage: request.Stage, StartDate: request.StartDate, DueDate: request.DueDate,
	})
	if errors.Is(err, contract.ErrInvalidIssue) {
		return writeError(ctx, http.StatusBadRequest, "invalid issue request")
	}
	if errors.Is(err, contract.ErrIssueNotFound) || errors.Is(err, contract.ErrProjectNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) || errors.Is(err, contract.ErrWorkspaceNotFound) {
		return writeError(ctx, http.StatusNotFound, "issue not found")
	}
	if err != nil || result.Issue == nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to create issue")
	}
	return ctx.JSON(http.StatusCreated, toPublicIssue(*result.Issue))
}

type publicIssue struct {
	ID            string         `json:"id"`
	WorkspaceID   string         `json:"workspace_id"`
	Number        int32          `json:"number"`
	Identifier    string         `json:"identifier"`
	Title         string         `json:"title"`
	Description   *string        `json:"description"`
	Status        string         `json:"status"`
	Priority      string         `json:"priority"`
	AssigneeType  *string        `json:"assignee_type"`
	AssigneeID    *string        `json:"assignee_id"`
	CreatorType   string         `json:"creator_type"`
	CreatorID     string         `json:"creator_id"`
	ParentIssueID *string        `json:"parent_issue_id"`
	ProjectID     *string        `json:"project_id"`
	Position      float64        `json:"position"`
	Stage         *int32         `json:"stage"`
	StartDate     *string        `json:"start_date"`
	DueDate       *string        `json:"due_date"`
	Metadata      map[string]any `json:"metadata"`
	Properties    map[string]any `json:"properties"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

func toPublicIssue(value contract.Issue) publicIssue {
	metadata, properties := value.Metadata, value.Properties
	if metadata == nil {
		metadata = map[string]any{}
	}
	if properties == nil {
		properties = map[string]any{}
	}
	return publicIssue{ID: value.Id, WorkspaceID: value.WorkspaceId, Number: value.Number, Identifier: value.Identifier, Title: value.Title, Description: value.Description, Status: value.Status, Priority: value.Priority, AssigneeType: value.AssigneeType, AssigneeID: value.AssigneeId, CreatorType: value.CreatorType, CreatorID: value.CreatorId, ParentIssueID: value.ParentIssueId, ProjectID: value.ProjectId, Position: value.Position, Stage: value.Stage, StartDate: value.StartDate, DueDate: value.DueDate, Metadata: metadata, Properties: properties, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (h *IssueReadHandler) requestIdentity(ctx kratoshttp.Context) (context.Context, string, error) {
	if h.identity == nil {
		return ctx.Request().Context(), "", contract.ErrWorkspaceActorRequired
	}
	identity, err := h.identity(ctx.Request())
	if err != nil {
		return ctx.Request().Context(), "", err
	}
	if identity.WorkspaceID == "" || identity.ActorID == "" {
		return ctx.Request().Context(), "", contract.ErrWorkspaceActorRequired
	}
	return contract.WithWorkspaceActor(ctx.Request().Context(), identity.ActorType, identity.ActorID), identity.WorkspaceID, nil
}

func (h *IssueReadHandler) get(ctx kratoshttp.Context) error {
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	result, err := h.service.GetIssue(requestContext, contract.GetIssueRequest{WorkspaceId: workspaceID, IssueId: ctx.Vars().Get("id")})
	if errors.Is(err, contract.ErrIssueNotFound) {
		return writeError(ctx, http.StatusNotFound, "issue not found")
	}
	if err != nil || result.Issue == nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to get issue")
	}
	return ctx.JSON(http.StatusOK, toPublicIssue(*result.Issue))
}

func (h *IssueReadHandler) list(ctx kratoshttp.Context) error {
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	return h.writeList(ctx, requestContext, workspaceID, queryParams(ctx.Request().URL.Query()))
}

func (h *IssueReadHandler) query(ctx kratoshttp.Context) error {
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return issueReadIdentityError(ctx, err)
	}
	var raw map[string]string
	if err := decodeJSON(ctx.Request().Body, &raw); err != nil {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	if !validIssueQueryKeys(raw) {
		return writeError(ctx, http.StatusBadRequest, "invalid request body")
	}
	return h.writeList(ctx, requestContext, workspaceID, raw)
}

func queryParams(values map[string][]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if len(value) > 0 {
			result[key] = value[0]
		}
	}
	return result
}

func (h *IssueReadHandler) writeList(ctx kratoshttp.Context, requestContext context.Context, workspaceID string, params map[string]string) error {
	issues, err := h.filteredIssues(requestContext, workspaceID, params)
	if err != nil {
		if errors.Is(err, errUnsupportedIssueQuery) {
			return writeError(ctx, http.StatusUnprocessableEntity, err.Error())
		}
		if errors.Is(err, contract.ErrActorOutsideWorkspace) {
			return writeError(ctx, http.StatusNotFound, "issue not found")
		}
		return writeError(ctx, http.StatusInternalServerError, "failed to list issues")
	}
	total := len(issues)
	offset := boundedInt(params["offset"], 0, total)
	limit := boundedInt(params["limit"], 50, 200)
	end := offset + limit
	if end > total {
		end = total
	}
	return ctx.JSON(http.StatusOK, map[string]any{"issues": issues[offset:end], "total": total})
}

func (h *IssueReadHandler) filteredIssues(ctx context.Context, workspaceID string, params map[string]string) ([]publicIssue, error) {
	if params["label_ids"] != "" || params["properties"] != "" || strings.HasPrefix(params["sort"], "property:") {
		return nil, fmt.Errorf("%w: label and property filters are not available", errUnsupportedIssueQuery)
	}
	if field := params["date_field"]; field != "" && field != "created_at" && field != "updated_at" {
		return nil, fmt.Errorf("%w: date field is not available", errUnsupportedIssueQuery)
	}
	if requested := strings.TrimSpace(params["workspace_id"]); requested != "" && requested != workspaceID {
		return nil, contract.ErrActorOutsideWorkspace
	}
	result, err := h.service.ListIssues(ctx, contract.ListIssuesRequest{WorkspaceId: workspaceID})
	if err != nil {
		return nil, err
	}
	statuses, priorities, ids := csvSet(params["statuses"]), csvSet(params["priorities"]), csvSet(params["ids"])
	assigneeIDs, assigneeTypes := csvSet(params["assignee_ids"]), csvSet(params["assignee_types"])
	assigneePairs, creatorPairs := map[string]struct{}{}, map[string]struct{}{}
	for _, actor := range strings.Split(params["assignee_filters"], ",") {
		parts := strings.SplitN(actor, ":", 2)
		if len(parts) == 2 {
			assigneePairs[parts[0]+"\x00"+parts[1]] = struct{}{}
		}
	}
	for _, actor := range strings.Split(params["creator_filters"], ",") {
		parts := strings.SplitN(actor, ":", 2)
		if len(parts) == 2 {
			creatorPairs[parts[0]+"\x00"+parts[1]] = struct{}{}
		}
	}
	projectIDs := csvSet(params["project_ids"])
	if params["assignee_id"] != "" {
		assigneeIDs[params["assignee_id"]] = struct{}{}
	}
	if params["project_id"] != "" {
		projectIDs[params["project_id"]] = struct{}{}
	}
	if params["status"] != "" {
		statuses[params["status"]] = struct{}{}
	}
	if params["priority"] != "" {
		priorities[params["priority"]] = struct{}{}
	}
	q := strings.ToLower(strings.TrimSpace(params["q"]))
	values := make([]publicIssue, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if len(statuses) > 0 && !setHas(statuses, issue.Status) {
			continue
		}
		if len(priorities) > 0 && !setHas(priorities, issue.Priority) {
			continue
		}
		if _, present := params["ids"]; present && !setHas(ids, issue.Id) {
			continue
		}
		if len(assigneeIDs) > 0 && ((issue.AssigneeId == nil && params["include_no_assignee"] != "true") || (issue.AssigneeId != nil && !setHas(assigneeIDs, *issue.AssigneeId))) {
			continue
		}
		if len(assigneeTypes) > 0 && (issue.AssigneeType == nil || !setHas(assigneeTypes, *issue.AssigneeType)) {
			continue
		}
		if len(assigneePairs) > 0 {
			if issue.AssigneeId == nil || issue.AssigneeType == nil {
				if params["include_no_assignee"] != "true" {
					continue
				}
			} else if !setHas(assigneePairs, *issue.AssigneeType+"\x00"+*issue.AssigneeId) {
				continue
			}
		} else if params["include_no_assignee"] == "true" && len(assigneeIDs) == 0 && issue.AssigneeId != nil {
			continue
		}
		if params["creator_id"] != "" && issue.CreatorId != params["creator_id"] {
			continue
		}
		if len(creatorPairs) > 0 && !setHas(creatorPairs, issue.CreatorType+"\x00"+issue.CreatorId) {
			continue
		}
		if actor := params["my_any_actor"]; actor != "" {
			parts := strings.SplitN(actor, ":", 2)
			assigned := len(parts) == 2 && issue.AssigneeType != nil && issue.AssigneeId != nil && *issue.AssigneeType == parts[0] && *issue.AssigneeId == parts[1]
			created := len(parts) == 2 && issue.CreatorType == parts[0] && issue.CreatorId == parts[1]
			if !assigned && !created {
				continue
			}
		}
		if len(projectIDs) > 0 && (issue.ProjectId == nil || !setHas(projectIDs, *issue.ProjectId)) && !(params["include_no_project"] == "true" && issue.ProjectId == nil) {
			continue
		}
		if len(projectIDs) == 0 && params["include_no_project"] == "true" && issue.ProjectId != nil {
			continue
		}
		if params["open_only"] == "true" && (issue.Status == "done" || issue.Status == "cancelled") {
			continue
		}
		if params["scheduled"] == "true" && issue.StartDate == nil && issue.DueDate == nil {
			continue
		}
		if !issueMatchesDate(issue, params["date_field"], params["date_start"], params["date_end"]) {
			continue
		}
		if !issueMatchesMetadata(issue.Metadata, params["metadata"]) {
			continue
		}
		if params["top_level_only"] == "true" && issue.ParentIssueId != nil {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(issue.Title), q) && strings.ToLower(issue.Identifier) != q && strconv.Itoa(int(issue.Number)) != q {
			continue
		}
		values = append(values, toPublicIssue(issue))
	}
	sortIssues(values, params["sort"], params["direction"])
	return values, nil
}

func issueMatchesDate(issue contract.Issue, field, start, end string) bool {
	if field == "" || (start == "" && end == "") {
		return true
	}
	var value string
	if field == "created_at" {
		value = issue.CreatedAt
	} else if field == "updated_at" {
		value = issue.UpdatedAt
	} else {
		return false
	}
	if start != "" && value < start {
		return false
	}
	if end != "" && value > end+"T23:59:59.999999999Z" {
		return false
	}
	return true
}

func issueMatchesMetadata(metadata map[string]any, raw string) bool {
	if raw == "" {
		return true
	}
	var expected map[string]any
	if json.Unmarshal([]byte(raw), &expected) != nil {
		return false
	}
	for key, value := range expected {
		actual, ok := metadata[key]
		if !ok || fmt.Sprint(actual) != fmt.Sprint(value) {
			return false
		}
	}
	return true
}

type tableQuery struct {
	Scope struct {
		Kind          string     `json:"kind"`
		ProjectID     string     `json:"project_id"`
		Actor         tableActor `json:"actor"`
		AssigneeTypes []string   `json:"assignee_types"`
		Relation      string     `json:"relation"`
	} `json:"scope"`
	Filters struct {
		Statuses          []string                           `json:"statuses"`
		Priorities        []string                           `json:"priorities"`
		IncludeSubIssues  *bool                              `json:"include_sub_issues"`
		Assignees         []tableActor                       `json:"assignees"`
		Creators          []tableActor                       `json:"creators"`
		ProjectIDs        []string                           `json:"project_ids"`
		LabelIDs          json.RawMessage                    `json:"label_ids"`
		Properties        json.RawMessage                    `json:"properties"`
		Date              struct{ Field, Start, End string } `json:"date"`
		IncludeNoAssignee bool                               `json:"include_no_assignee"`
		IncludeNoProject  bool                               `json:"include_no_project"`
	} `json:"filters"`
	Search string                            `json:"search"`
	Sort   struct{ Field, Direction string } `json:"sort"`
}
type tableActor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
type tableGroup struct {
	Kind       string `json:"kind"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	PropertyID string `json:"property_id"`
}
type tablePage struct {
	Limit  *int    `json:"limit"`
	Cursor *string `json:"cursor"`
}
type tableRequest struct {
	Query     tableQuery `json:"query"`
	Group     tableGroup `json:"group"`
	GroupKey  *string    `json:"group_key"`
	Hierarchy struct {
		Enabled bool `json:"enabled"`
	} `json:"hierarchy"`
	ParentID     *string                             `json:"parent_id"`
	Page         tablePage                           `json:"page"`
	Facets       []struct{ Kind, PropertyID string } `json:"facets"`
	IncludeTotal *bool                               `json:"include_total"`
}

type issueCursor struct {
	Kind, Workspace, Fingerprint, Group, GroupKey, Parent, Last string
	Hierarchy                                                   bool
}

func (h *IssueReadHandler) tableIssues(ctx kratoshttp.Context, request *tableRequest) (context.Context, string, []publicIssue, error) {
	requestContext, workspaceID, err := h.requestIdentity(ctx)
	if err != nil {
		return nil, "", nil, err
	}
	if request.Query.Scope.Kind != "workspace" && request.Query.Scope.Kind != "project" && request.Query.Scope.Kind != "assignee" && request.Query.Scope.Kind != "creator" && request.Query.Scope.Kind != "my" {
		return nil, "", nil, fmt.Errorf("unsupported scope")
	}
	filters := request.Query.Filters
	if hasJSONValue(filters.LabelIDs) || hasJSONValue(filters.Properties) {
		return nil, "", nil, fmt.Errorf("unsupported filter")
	}
	if field := request.Query.Sort.Field; field != "" && field != "position" && field != "title" && field != "created_at" && field != "updated_at" && field != "status" && field != "priority" && field != "start_date" && field != "due_date" {
		return nil, "", nil, fmt.Errorf("unsupported sort")
	}
	if request.Query.Sort.Direction != "asc" && request.Query.Sort.Direction != "desc" {
		return nil, "", nil, fmt.Errorf("invalid sort direction")
	}
	params := map[string]string{"statuses": strings.Join(request.Query.Filters.Statuses, ","), "priorities": strings.Join(request.Query.Filters.Priorities, ","), "q": request.Query.Search, "sort": request.Query.Sort.Field, "direction": request.Query.Sort.Direction}
	params["assignee_filters"] = actorList(filters.Assignees)
	params["creator_filters"] = actorList(filters.Creators)
	params["project_ids"] = strings.Join(filters.ProjectIDs, ",")
	if filters.IncludeNoAssignee {
		params["include_no_assignee"] = "true"
	}
	if filters.IncludeNoProject {
		params["include_no_project"] = "true"
	}
	params["date_field"], params["date_start"], params["date_end"] = filters.Date.Field, filters.Date.Start, filters.Date.End
	if request.Query.Scope.Kind == "workspace" {
		params["assignee_types"] = strings.Join(request.Query.Scope.AssigneeTypes, ",")
	}
	switch request.Query.Scope.Kind {
	case "project":
		params["project_id"] = request.Query.Scope.ProjectID
	case "assignee":
		params["assignee_filters"] = request.Query.Scope.Actor.Type + ":" + request.Query.Scope.Actor.ID
	case "creator":
		params["creator_filters"] = request.Query.Scope.Actor.Type + ":" + request.Query.Scope.Actor.ID
	case "my":
		actor, ok := contract.WorkspaceActorFromContext(requestContext)
		if !ok {
			return nil, "", nil, contract.ErrWorkspaceActorRequired
		}
		pair := actor.Type + ":" + actor.ID
		switch request.Query.Scope.Relation {
		case "assigned":
			params["assignee_filters"] = pair
		case "created":
			params["creator_filters"] = pair
		case "any":
			params["my_any_actor"] = pair
		default:
			return nil, "", nil, fmt.Errorf("unsupported my relation")
		}
	}
	if request.Query.Filters.IncludeSubIssues != nil && !*request.Query.Filters.IncludeSubIssues {
		params["top_level_only"] = "true"
	}
	issues, err := h.filteredIssues(requestContext, workspaceID, params)
	return requestContext, workspaceID, issues, err
}

func actorList(actors []tableActor) string {
	values := make([]string, 0, len(actors))
	for _, actor := range actors {
		values = append(values, actor.Type+":"+actor.ID)
	}
	return strings.Join(values, ",")
}

func issueGroupDescriptor(issue publicIssue, kind string, issueByID map[string]publicIssue) (string, map[string]any) {
	switch kind {
	case "status":
		return "status:" + issue.Status, map[string]any{"kind": "status", "status": issue.Status}
	case "assignee":
		if issue.AssigneeType == nil || issue.AssigneeID == nil {
			return "assignee:unassigned", map[string]any{"kind": "assignee", "actor": nil}
		}
		return "assignee:" + *issue.AssigneeType + ":" + *issue.AssigneeID, map[string]any{"kind": "assignee", "actor": map[string]any{"type": *issue.AssigneeType, "id": *issue.AssigneeID}}
	case "project":
		if issue.ProjectID == nil {
			return "project:none", map[string]any{"kind": "project", "project_id": nil}
		}
		return "project:" + *issue.ProjectID, map[string]any{"kind": "project", "project_id": *issue.ProjectID}
	default:
		if issue.ParentIssueID == nil {
			return "parent:none", map[string]any{"kind": "parent", "parent_id": nil, "parent": nil, "value_state": "unset"}
		}
		parent, ok := issueByID[*issue.ParentIssueID]
		if !ok {
			return "parent:" + *issue.ParentIssueID, map[string]any{"kind": "parent", "parent_id": *issue.ParentIssueID, "parent": nil, "value_state": "unavailable"}
		}
		return "parent:" + *issue.ParentIssueID, map[string]any{"kind": "parent", "parent_id": *issue.ParentIssueID, "parent": map[string]any{"id": parent.ID, "number": parent.Number, "identifier": parent.Identifier, "title": parent.Title, "status": parent.Status}, "value_state": "value"}
	}
}

func (h *IssueReadHandler) facets(ctx kratoshttp.Context) error {
	var request tableRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, 400, "invalid request body")
	}
	_, _, issues, err := h.tableIssues(ctx, &request)
	if err != nil {
		return tableError(ctx, err)
	}
	result := make([]map[string]any, 0, len(request.Facets))
	for _, facet := range request.Facets {
		if facet.Kind != "status" && facet.Kind != "priority" && facet.Kind != "assignee" && facet.Kind != "creator" && facet.Kind != "project" {
			return writeError(ctx, 422, "unsupported issue table facet")
		}
		counts := map[string]int{}
		for _, issue := range issues {
			if facet.Kind == "status" {
				counts[issue.Status]++
			} else if facet.Kind == "priority" {
				counts[issue.Priority]++
			} else if facet.Kind == "assignee" {
				key := "unassigned"
				if issue.AssigneeType != nil && issue.AssigneeID != nil {
					key = *issue.AssigneeType + ":" + *issue.AssigneeID
				}
				counts[key]++
			} else if facet.Kind == "creator" {
				counts[issue.CreatorType+":"+issue.CreatorID]++
			} else {
				key := "none"
				if issue.ProjectID != nil {
					key = *issue.ProjectID
				}
				counts[key]++
			}
		}
		keys := make([]string, 0, len(counts))
		for key := range counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		values := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			values = append(values, map[string]any{"key": key, "count": counts[key]})
		}
		result = append(result, map[string]any{"kind": facet.Kind, "values": values})
	}
	total := 0
	if request.IncludeTotal == nil || *request.IncludeTotal {
		total = len(issues)
	}
	return ctx.JSON(200, map[string]any{"query_fingerprint": fingerprint(request.Query), "total": total, "facets": result})
}

func (h *IssueReadHandler) groups(ctx kratoshttp.Context) error {
	var request tableRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, 400, "invalid request body")
	}
	if request.Group.Kind != "status" && request.Group.Kind != "assignee" && request.Group.Kind != "project" && request.Group.Kind != "parent" {
		return writeError(ctx, 422, "unsupported issue table group")
	}
	_, workspaceID, issues, err := h.tableIssues(ctx, &request)
	if err != nil {
		return tableError(ctx, err)
	}
	issueByID := make(map[string]publicIssue, len(issues))
	for _, issue := range issues {
		issueByID[issue.ID] = issue
	}
	descriptors := map[string]map[string]any{}
	for _, issue := range issues {
		key, value := issueGroupDescriptor(issue, request.Group.Kind, issueByID)
		group := descriptors[key]
		if group == nil {
			group = map[string]any{"key": key, "value": value, "count": 0}
			descriptors[key] = group
		}
		group["count"] = group["count"].(int) + 1
	}
	keys := make([]string, 0, len(descriptors))
	for key := range descriptors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	groups := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		groups = append(groups, descriptors[key])
	}
	fp := fingerprint(request.Query)
	offset := 0
	if request.Page.Cursor != nil {
		cursor, decodeErr := decodeIssueCursor(*request.Page.Cursor)
		if decodeErr != nil {
			return writeError(ctx, 400, "invalid issue table cursor")
		}
		if cursor.Kind != "groups" || cursor.Workspace != workspaceID || cursor.Fingerprint != fp || cursor.Group != request.Group.Kind {
			return writeError(ctx, 409, "cursor_query_mismatch")
		}
		for index := range groups {
			if groups[index]["key"] == cursor.Last {
				offset = index + 1
				break
			}
		}
		if offset == 0 {
			return writeError(ctx, 409, "cursor_query_mismatch")
		}
	}
	limit, limitErr := tablePageLimit(request.Page)
	if limitErr != nil {
		return writeError(ctx, 400, limitErr.Error())
	}
	end := offset + limit
	if end > len(groups) {
		end = len(groups)
	}
	var next any
	if end < len(groups) {
		next = encodeIssueCursor(issueCursor{Kind: "groups", Workspace: workspaceID, Fingerprint: fp, Group: request.Group.Kind, Last: groups[end-1]["key"].(string)})
	}
	return ctx.JSON(200, map[string]any{"query_fingerprint": fp, "total": len(issues), "groups": groups[offset:end], "next_cursor": next})
}

func (h *IssueReadHandler) rows(ctx kratoshttp.Context) error {
	var request tableRequest
	if err := decodeJSON(ctx.Request().Body, &request); err != nil {
		return writeError(ctx, 400, "invalid request body")
	}
	if request.Group.Kind != "none" && request.Group.Kind != "status" && request.Group.Kind != "assignee" && request.Group.Kind != "project" && request.Group.Kind != "parent" {
		return writeError(ctx, 422, "unsupported issue table group")
	}
	_, workspaceID, baseIssues, err := h.tableIssues(ctx, &request)
	if err != nil {
		return tableError(ctx, err)
	}
	issues := baseIssues
	if request.Group.Kind != "none" {
		if request.GroupKey == nil {
			return writeError(ctx, 422, "invalid issue table group key")
		}
		issueByID := make(map[string]publicIssue, len(issues))
		for _, issue := range issues {
			issueByID[issue.ID] = issue
		}
		filtered := issues[:0]
		for _, issue := range issues {
			key, _ := issueGroupDescriptor(issue, request.Group.Kind, issueByID)
			if key == *request.GroupKey {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}
	// Counts are derived from the same group branch used by expansion. A
	// status:todo parent must not advertise a done child that the matching
	// status:todo expansion will not return.
	childCounts := map[string]int{}
	for _, issue := range issues {
		if issue.ParentIssueID != nil {
			childCounts[*issue.ParentIssueID]++
		}
	}
	if request.Hierarchy.Enabled && request.ParentID == nil {
		filtered := issues[:0]
		for _, issue := range issues {
			if issue.ParentIssueID == nil {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	} else if request.ParentID != nil {
		filtered := issues[:0]
		for _, issue := range issues {
			if issue.ParentIssueID != nil && *issue.ParentIssueID == *request.ParentID {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}
	branchTotal := len(issues)
	fp := fingerprint(request.Query)
	offset := 0
	if request.Page.Cursor != nil {
		cursor, decodeErr := decodeIssueCursor(*request.Page.Cursor)
		if decodeErr != nil {
			return writeError(ctx, 400, "invalid issue table cursor")
		}
		binding := rowCursorBinding(workspaceID, fp, request)
		if cursor.Kind != binding.Kind || cursor.Workspace != binding.Workspace || cursor.Fingerprint != binding.Fingerprint || cursor.Group != binding.Group || cursor.GroupKey != binding.GroupKey || cursor.Hierarchy != binding.Hierarchy || cursor.Parent != binding.Parent {
			return writeError(ctx, 409, "cursor_query_mismatch")
		}
		for index := range issues {
			if issues[index].ID == cursor.Last {
				offset = index + 1
				break
			}
		}
		if offset == 0 {
			return writeError(ctx, 409, "cursor_query_mismatch")
		}
	}
	limit, limitErr := tablePageLimit(request.Page)
	if limitErr != nil {
		return writeError(ctx, 400, limitErr.Error())
	}
	end := offset + limit
	if end > branchTotal {
		end = branchTotal
	}
	rows := make([]map[string]any, 0, end-offset)
	for _, issue := range issues[offset:end] {
		rows = append(rows, map[string]any{"issue": issue, "direct_child_count": childCounts[issue.ID]})
	}
	var next any
	if end < branchTotal {
		binding := rowCursorBinding(workspaceID, fp, request)
		binding.Last = issues[end-1].ID
		next = encodeIssueCursor(binding)
	}
	total := 0
	if request.Page.Cursor == nil && request.Group.Kind == "none" && request.ParentID == nil {
		total = len(baseIssues)
	}
	return ctx.JSON(200, map[string]any{"query_fingerprint": fp, "group_key": request.GroupKey, "parent_id": request.ParentID, "total": total, "rows": rows, "branch_total": len(rows), "next_cursor": next})
}

func tablePageLimit(page tablePage) (int, error) {
	if page.Limit == nil {
		return 50, nil
	}
	if *page.Limit < 1 || *page.Limit > 100 {
		return 0, errors.New("page limit must be between 1 and 100")
	}
	return *page.Limit, nil
}

func decodeJSON(body io.Reader, target any) error {
	const maximumJSONBody = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(body, maximumJSONBody+1))
	if err != nil || len(raw) > maximumJSONBody {
		return errors.New("request body too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}

func validIssueQueryKeys(params map[string]string) bool {
	allowed := map[string]struct{}{
		"limit": {}, "offset": {}, "workspace_id": {}, "q": {}, "status": {}, "statuses": {},
		"priority": {}, "priorities": {}, "assignee_id": {}, "assignee_ids": {}, "assignee_types": {},
		"assignee_filters": {}, "include_no_assignee": {}, "creator_id": {}, "creator_filters": {},
		"project_id": {}, "project_ids": {}, "include_no_project": {}, "ids": {}, "label_ids": {},
		"metadata": {}, "properties": {}, "open_only": {}, "scheduled": {}, "date_field": {},
		"date_start": {}, "date_end": {}, "sort": {}, "direction": {}, "top_level_only": {},
	}
	for key := range params {
		if _, ok := allowed[key]; !ok {
			return false
		}
	}
	return true
}
func fingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:8])
}
func hasJSONValue(raw json.RawMessage) bool {
	value := strings.TrimSpace(string(raw))
	return value != "" && value != "null" && value != "[]" && value != "{}"
}
func encodeIssueCursor(cursor issueCursor) string {
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}
func decodeIssueCursor(raw string) (issueCursor, error) {
	var cursor issueCursor
	encoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursor, err
	}
	err = json.Unmarshal(encoded, &cursor)
	if err != nil || cursor.Kind == "" || cursor.Last == "" {
		return cursor, errors.New("invalid cursor")
	}
	return cursor, nil
}
func rowCursorBinding(workspaceID, fp string, request tableRequest) issueCursor {
	binding := issueCursor{Kind: "rows", Workspace: workspaceID, Fingerprint: fp, Group: request.Group.Kind, Hierarchy: request.Hierarchy.Enabled}
	if request.GroupKey != nil {
		binding.GroupKey = *request.GroupKey
	}
	if request.ParentID != nil {
		binding.Parent = *request.ParentID
	}
	return binding
}
func csvSet(value string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result[item] = struct{}{}
		}
	}
	return result
}
func setHas(set map[string]struct{}, key string) bool { _, ok := set[key]; return ok }
func boundedInt(raw string, fallback, maximum int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}
func sortIssues(values []publicIssue, field, direction string) {
	sort.SliceStable(values, func(i, j int) bool {
		leftIssue, rightIssue := values[i], values[j]
		comparison := 0
		if field == "start_date" || field == "due_date" {
			left, right := leftIssue.StartDate, rightIssue.StartDate
			if field == "due_date" {
				left, right = leftIssue.DueDate, rightIssue.DueDate
			}
			if left == nil && right != nil {
				return false
			}
			if left != nil && right == nil {
				return true
			}
			if left != nil && right != nil {
				comparison = strings.Compare(*left, *right)
			}
		} else if field == "" || field == "position" {
			if leftIssue.Position < rightIssue.Position {
				comparison = -1
			} else if leftIssue.Position > rightIssue.Position {
				comparison = 1
			}
		} else if field == "status" {
			comparison = compareRank(statusRank(leftIssue.Status), statusRank(rightIssue.Status))
		} else if field == "priority" {
			comparison = compareRank(priorityRank(leftIssue.Priority), priorityRank(rightIssue.Priority))
		} else {
			left, right := leftIssue.Title, rightIssue.Title
			if field == "created_at" {
				left, right = leftIssue.CreatedAt, rightIssue.CreatedAt
			}
			if field == "updated_at" {
				left, right = leftIssue.UpdatedAt, rightIssue.UpdatedAt
			}
			comparison = strings.Compare(left, right)
		}
		if comparison != 0 {
			if direction == "desc" && field != "position" && field != "" {
				return comparison > 0
			}
			return comparison < 0
		}
		if leftIssue.CreatedAt != rightIssue.CreatedAt {
			return leftIssue.CreatedAt > rightIssue.CreatedAt
		}
		return leftIssue.ID > rightIssue.ID
	})
}
func compareRank(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
func statusRank(value string) int {
	for i, item := range []string{"backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled"} {
		if value == item {
			return i
		}
	}
	return 99
}
func priorityRank(value string) int {
	for i, item := range []string{"urgent", "high", "medium", "low", "none"} {
		if value == item {
			return i
		}
	}
	return 99
}
func tableError(ctx kratoshttp.Context, err error) error {
	if strings.Contains(err.Error(), "invalid") {
		return writeError(ctx, 400, err.Error())
	}
	if strings.Contains(err.Error(), "unsupported") {
		return writeError(ctx, 422, err.Error())
	}
	return issueReadIdentityError(ctx, err)
}
func issueReadIdentityError(ctx kratoshttp.Context, err error) error {
	if errors.Is(err, contract.ErrWorkspaceNotFound) || errors.Is(err, contract.ErrActorOutsideWorkspace) {
		return writeError(ctx, 404, "issue not found")
	}
	if errors.Is(err, contract.ErrWorkspaceActorRequired) || strings.Contains(strings.ToLower(err.Error()), "session") || strings.Contains(strings.ToLower(err.Error()), "authenticated") || strings.Contains(strings.ToLower(err.Error()), "token") {
		return writeError(ctx, 401, "user not authenticated")
	}
	return writeError(ctx, 500, "issue operation failed")
}
