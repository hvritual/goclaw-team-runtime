package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultListLimit     = 100
	maxListLimit         = 500
	defaultSearchLimit   = 20
	maxSearchLimit       = 100
	defaultTraverseDepth = 2
	maxTraverseDepth     = 4
	defaultTraverseNodes = 64
	maxTraverseNodes     = 256
)

type Dependencies struct {
	Service  contract.Service
	Compiler contract.AuthorizedContextCompiler
	UserID   string
	Version  string
}

type toolset struct {
	service  contract.Service
	compiler contract.AuthorizedContextCompiler
	actor    contract.Actor
}

func New(deps Dependencies) (*mcpsdk.Server, error) {
	deps.UserID = strings.TrimSpace(deps.UserID)
	deps.Version = strings.TrimSpace(deps.Version)
	if deps.Service == nil || deps.Compiler == nil || deps.UserID == "" {
		return nil, errors.New("engineering MCP service, compiler, and fixed user identity are required")
	}
	if deps.Version == "" {
		deps.Version = "dev"
	}
	tools := &toolset{service: deps.Service, compiler: deps.Compiler, actor: contract.Actor{UserID: deps.UserID}}
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "goclaw-engineering", Version: deps.Version}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "engineering_entity_get", Description: "Get one EngineeringEntity in an authorized workspace."}, tools.entityGet)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "engineering_entity_list", Description: "List EngineeringEntities with deterministic type/status filtering."}, tools.entityList)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "engineering_entity_search", Description: "Search EngineeringEntities by canonical id, name, type, status, or owner reference."}, tools.entitySearch)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "engineering_thread_traverse", Description: "Traverse the bounded Engineering Thread graph from one canonical node."}, tools.threadTraverse)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "engineering_change_get", Description: "Get one Engineering Change without accepting or mutating it."}, tools.changeGet)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "engineering_change_list", Description: "List Engineering Changes, optionally filtered by affected entity and status."}, tools.changeList)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "context_pack_get", Description: "Read one immutable frozen ContextPack."}, tools.contextPackGet)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "context_pack_compile", Description: "Compile and freeze a deterministic ContextPack under owner/admin authorization. This cannot accept Changes, publish Knowledge, change permissions, or satisfy DoneGate."}, tools.contextPackCompile)
	return server, nil
}

type entityGetInput struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"authorized workspace id"`
	EntityID    string `json:"entity_id" jsonschema:"canonical EngineeringEntity id"`
}
type entityGetOutput struct {
	Entity contract.Entity `json:"entity"`
}

func (t *toolset) entityGet(ctx context.Context, _ *mcpsdk.CallToolRequest, input entityGetInput) (*mcpsdk.CallToolResult, entityGetOutput, error) {
	value, err := t.service.GetEntity(ctx, t.actor, input.WorkspaceID, input.EntityID)
	if err != nil {
		return nil, entityGetOutput{}, stableToolError(err)
	}
	return nil, entityGetOutput{Entity: value}, nil
}

type entityListInput struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"authorized workspace id"`
	Type        string `json:"type,omitempty" jsonschema:"optional canonical entity type"`
	Status      string `json:"status,omitempty" jsonschema:"optional entity status"`
	Limit       int    `json:"limit,omitempty" jsonschema:"maximum results, default 100, max 500"`
}
type entityListOutput struct {
	Items []contract.Entity `json:"items"`
}

func (t *toolset) entityList(ctx context.Context, _ *mcpsdk.CallToolRequest, input entityListInput) (*mcpsdk.CallToolResult, entityListOutput, error) {
	values, err := t.filteredEntities(ctx, input.WorkspaceID, "", input.Type, input.Status, input.Limit, defaultListLimit, maxListLimit)
	if err != nil {
		return nil, entityListOutput{}, err
	}
	return nil, entityListOutput{Items: values}, nil
}

type entitySearchInput struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"authorized workspace id"`
	Query       string `json:"query" jsonschema:"case-insensitive search text"`
	Type        string `json:"type,omitempty" jsonschema:"optional canonical entity type"`
	Status      string `json:"status,omitempty" jsonschema:"optional entity status"`
	Limit       int    `json:"limit,omitempty" jsonschema:"maximum results, default 20, max 100"`
}
type entitySearchOutput struct {
	Items []contract.Entity `json:"items"`
}

func (t *toolset) entitySearch(ctx context.Context, _ *mcpsdk.CallToolRequest, input entitySearchInput) (*mcpsdk.CallToolResult, entitySearchOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, entitySearchOutput{}, errors.New("invalid_argument: query is required")
	}
	values, err := t.filteredEntities(ctx, input.WorkspaceID, query, input.Type, input.Status, input.Limit, defaultSearchLimit, maxSearchLimit)
	if err != nil {
		return nil, entitySearchOutput{}, err
	}
	return nil, entitySearchOutput{Items: values}, nil
}

func (t *toolset) filteredEntities(ctx context.Context, workspaceID, query, entityType, status string, limit, defaultLimit, maxLimit int) ([]contract.Entity, error) {
	resolvedLimit, err := boundedLimit(limit, defaultLimit, maxLimit)
	if err != nil {
		return nil, err
	}
	values, err := t.service.ListEntities(ctx, t.actor, workspaceID)
	if err != nil {
		return nil, stableToolError(err)
	}
	query = strings.ToLower(strings.TrimSpace(query))
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	status = strings.ToLower(strings.TrimSpace(status))
	result := make([]contract.Entity, 0, min(len(values), resolvedLimit))
	for _, value := range values {
		if entityType != "" && strings.ToLower(value.Type) != entityType {
			continue
		}
		if status != "" && strings.ToLower(value.Status) != status {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(strings.Join([]string{value.ID, value.Name, value.Type, value.Status, value.OwnerRef}, "\x00"))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if len(result) > resolvedLimit {
		result = result[:resolvedLimit]
	}
	return result, nil
}

type threadTraverseInput struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"authorized workspace id"`
	NodeKind    string `json:"node_kind,omitempty" jsonschema:"canonical node kind; defaults to engineering_entity"`
	NodeID      string `json:"node_id" jsonschema:"starting canonical node id"`
	MaxDepth    int    `json:"max_depth,omitempty" jsonschema:"BFS depth, default 2, max 4"`
	MaxNodes    int    `json:"max_nodes,omitempty" jsonschema:"node budget, default 64, max 256"`
}
type traversedNode struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Depth int    `json:"depth"`
}
type threadTraverseOutput struct {
	Nodes     []traversedNode       `json:"nodes"`
	Edges     []contract.ThreadEdge `json:"edges"`
	Truncated bool                  `json:"truncated"`
}

func (t *toolset) threadTraverse(ctx context.Context, _ *mcpsdk.CallToolRequest, input threadTraverseInput) (*mcpsdk.CallToolResult, threadTraverseOutput, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	nodeKind := strings.TrimSpace(input.NodeKind)
	if nodeKind == "" {
		nodeKind = "engineering_entity"
	}
	nodeID := strings.TrimSpace(input.NodeID)
	if workspaceID == "" || nodeID == "" {
		return nil, threadTraverseOutput{}, errors.New("invalid_argument: workspace_id and node_id are required")
	}
	maxDepth := input.MaxDepth
	if maxDepth == 0 {
		maxDepth = defaultTraverseDepth
	}
	maxNodes := input.MaxNodes
	if maxNodes == 0 {
		maxNodes = defaultTraverseNodes
	}
	if maxDepth < 0 || maxDepth > maxTraverseDepth || maxNodes < 1 || maxNodes > maxTraverseNodes {
		return nil, threadTraverseOutput{}, errors.New("invalid_argument: traversal limits are outside supported bounds")
	}
	type queuedNode struct {
		ref   contract.NodeRef
		depth int
	}
	start := contract.NodeRef{Kind: nodeKind, ID: nodeID}
	queue := []queuedNode{{ref: start}}
	visited := map[string]int{nodeKey(start): 0}
	edges := map[string]contract.ThreadEdge{}
	truncated := false
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.depth >= maxDepth {
			continue
		}
		values, err := t.service.ListThreadEdges(ctx, t.actor, workspaceID, current.ref)
		if err != nil {
			return nil, threadTraverseOutput{}, stableToolError(err)
		}
		sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
		for _, edge := range values {
			edges[edge.ID] = edge
			neighbor, ok := otherNode(edge, current.ref)
			if !ok {
				continue
			}
			key := nodeKey(neighbor)
			if _, seen := visited[key]; seen {
				continue
			}
			if len(visited) >= maxNodes {
				truncated = true
				continue
			}
			visited[key] = current.depth + 1
			queue = append(queue, queuedNode{ref: neighbor, depth: current.depth + 1})
		}
	}
	nodes := make([]traversedNode, 0, len(visited))
	for key, depth := range visited {
		kind, id, _ := strings.Cut(key, "\x00")
		nodes = append(nodes, traversedNode{Kind: kind, ID: id, Depth: depth})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Depth != nodes[j].Depth {
			return nodes[i].Depth < nodes[j].Depth
		}
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].ID < nodes[j].ID
	})
	edgeList := make([]contract.ThreadEdge, 0, len(edges))
	for _, edge := range edges {
		edgeList = append(edgeList, edge)
	}
	sort.Slice(edgeList, func(i, j int) bool { return edgeList[i].ID < edgeList[j].ID })
	return nil, threadTraverseOutput{Nodes: nodes, Edges: edgeList, Truncated: truncated}, nil
}

func otherNode(edge contract.ThreadEdge, current contract.NodeRef) (contract.NodeRef, bool) {
	if sameNode(edge.From, current) {
		return edge.To, true
	}
	if sameNode(edge.To, current) {
		return edge.From, true
	}
	return contract.NodeRef{}, false
}

func sameNode(left, right contract.NodeRef) bool {
	return left.Kind == right.Kind && left.ID == right.ID
}
func nodeKey(value contract.NodeRef) string { return value.Kind + "\x00" + value.ID }

type changeGetInput struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"authorized workspace id"`
	ChangeID    string `json:"change_id" jsonschema:"Engineering Change id"`
}
type changeGetOutput struct {
	Change contract.Change `json:"change"`
}

func (t *toolset) changeGet(ctx context.Context, _ *mcpsdk.CallToolRequest, input changeGetInput) (*mcpsdk.CallToolResult, changeGetOutput, error) {
	value, err := t.service.GetChange(ctx, t.actor, input.WorkspaceID, input.ChangeID)
	if err != nil {
		return nil, changeGetOutput{}, stableToolError(err)
	}
	return nil, changeGetOutput{Change: value}, nil
}

type changeListInput struct {
	WorkspaceID      string `json:"workspace_id" jsonschema:"authorized workspace id"`
	AffectedEntityID string `json:"affected_entity_id,omitempty" jsonschema:"optional affected EngineeringEntity id"`
	Status           string `json:"status,omitempty" jsonschema:"optional Change status"`
	Limit            int    `json:"limit,omitempty" jsonschema:"maximum results, default 100, max 500"`
}
type changeListOutput struct {
	Items []contract.Change `json:"items"`
}

func (t *toolset) changeList(ctx context.Context, _ *mcpsdk.CallToolRequest, input changeListInput) (*mcpsdk.CallToolResult, changeListOutput, error) {
	limit, err := boundedLimit(input.Limit, defaultListLimit, maxListLimit)
	if err != nil {
		return nil, changeListOutput{}, err
	}
	values, err := t.service.ListChanges(ctx, t.actor, input.WorkspaceID, input.AffectedEntityID)
	if err != nil {
		return nil, changeListOutput{}, stableToolError(err)
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	result := make([]contract.Change, 0, min(len(values), limit))
	for _, value := range values {
		if status != "" && strings.ToLower(value.Status) != status {
			continue
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].UpdatedAt.After(result[j].UpdatedAt)
		}
		return result[i].ID < result[j].ID
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return nil, changeListOutput{Items: result}, nil
}

type contextPackGetInput struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"authorized workspace id"`
	PackID      string `json:"pack_id" jsonschema:"immutable ContextPack id"`
}
type contextPackGetOutput struct {
	Pack contract.ContextPack `json:"pack"`
}

func (t *toolset) contextPackGet(ctx context.Context, _ *mcpsdk.CallToolRequest, input contextPackGetInput) (*mcpsdk.CallToolResult, contextPackGetOutput, error) {
	value, err := t.service.GetContextPack(ctx, t.actor, input.WorkspaceID, input.PackID)
	if err != nil {
		return nil, contextPackGetOutput{}, stableToolError(err)
	}
	return nil, contextPackGetOutput{Pack: value}, nil
}

type contextPackCompileInput struct {
	WorkspaceID           string `json:"workspace_id" jsonschema:"authorized workspace id"`
	PackID                string `json:"pack_id" jsonschema:"new immutable ContextPack id"`
	WorkItemKind          string `json:"work_item_kind" jsonschema:"project, requirement, issue, todo, or task"`
	WorkItemID            string `json:"work_item_id" jsonschema:"canonical work item id"`
	WorkItemRevision      string `json:"work_item_revision" jsonschema:"frozen work item revision"`
	PolicyVersion         string `json:"policy_version" jsonschema:"context selection policy version"`
	MaxDepth              int    `json:"max_depth,omitempty"`
	MaxEntities           int    `json:"max_entities,omitempty"`
	SourceStaleSeconds    int64  `json:"source_stale_seconds,omitempty"`
	KnowledgeStaleSeconds int64  `json:"knowledge_stale_seconds,omitempty"`
	MaxReferences         int    `json:"max_references,omitempty"`
	MaxEstimatedTokens    int    `json:"max_estimated_tokens,omitempty"`
	MaxRecentChanges      int    `json:"max_recent_changes,omitempty"`
}
type contextPackCompileOutput struct {
	Result contract.CompileContextResult `json:"result"`
}

func (t *toolset) contextPackCompile(ctx context.Context, _ *mcpsdk.CallToolRequest, input contextPackCompileInput) (*mcpsdk.CallToolResult, contextPackCompileOutput, error) {
	if input.SourceStaleSeconds < 0 || input.KnowledgeStaleSeconds < 0 {
		return nil, contextPackCompileOutput{}, errors.New("invalid_argument: stale windows cannot be negative")
	}
	request := contract.CompileContextRequest{
		WorkspaceID:      strings.TrimSpace(input.WorkspaceID),
		PackID:           strings.TrimSpace(input.PackID),
		WorkItem:         contract.NodeRef{Kind: strings.TrimSpace(input.WorkItemKind), ID: strings.TrimSpace(input.WorkItemID)},
		WorkItemRevision: strings.TrimSpace(input.WorkItemRevision),
		Policy: contract.ContextCompilePolicy{
			Version: strings.TrimSpace(input.PolicyVersion), MaxDepth: input.MaxDepth, MaxEntities: input.MaxEntities,
			SourceStaleAfter:    time.Duration(input.SourceStaleSeconds) * time.Second,
			KnowledgeStaleAfter: time.Duration(input.KnowledgeStaleSeconds) * time.Second,
			MaxReferences:       input.MaxReferences, MaxEstimatedTokens: input.MaxEstimatedTokens, MaxRecentChanges: input.MaxRecentChanges,
		},
	}
	value, err := t.compiler.CompileContext(ctx, t.actor, request)
	if err != nil {
		return nil, contextPackCompileOutput{}, stableToolError(err)
	}
	return nil, contextPackCompileOutput{Result: value}, nil
}

func boundedLimit(value, defaultValue, maxValue int) (int, error) {
	if value == 0 {
		return defaultValue, nil
	}
	if value < 1 || value > maxValue {
		return 0, fmt.Errorf("invalid_argument: limit must be between 1 and %d", maxValue)
	}
	return value, nil
}

func stableToolError(err error) error {
	switch {
	case errors.Is(err, contract.ErrInvalidArgument):
		return errors.New("invalid_argument")
	case errors.Is(err, contract.ErrForbidden):
		return errors.New("forbidden")
	case errors.Is(err, contract.ErrNotFound):
		return errors.New("not_found")
	case errors.Is(err, contract.ErrConflict):
		return errors.New("conflict")
	case errors.Is(err, contract.ErrUnavailable):
		return errors.New("unavailable")
	default:
		return errors.New("operation_failed")
	}
}
