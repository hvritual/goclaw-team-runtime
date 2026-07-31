package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const (
	scopeKnowledgeRead          = "knowledge:read"
	scopeKnowledgeCandidateRead = "knowledge:candidate:read"
	scopeKnowledgePropose       = "knowledge:propose"
)

type mcpKnowledgeSearchInput struct {
	Query             string `json:"query,omitempty" jsonschema:"Text to search for in published knowledge."`
	ProjectID         string `json:"project_id,omitempty" jsonschema:"Optional Multica project ID."`
	Kind              string `json:"kind,omitempty" jsonschema:"Optional knowledge kind filter."`
	IncludeCandidates bool   `json:"include_candidates,omitempty" jsonschema:"Include reviewable candidates when authorized."`
	Limit             int    `json:"limit,omitempty" jsonschema:"Maximum number of results, from 1 to 100."`
	Cursor            string `json:"cursor,omitempty" jsonschema:"Published knowledge pagination cursor."`
	CandidateCursor   string `json:"candidate_cursor,omitempty" jsonschema:"Independent candidate pagination cursor."`
}

type mcpKnowledgeGetInput struct {
	KnowledgeID string `json:"knowledge_id" jsonschema:"Published knowledge entry ID."`
}

type mcpKnowledgeProposeInput struct {
	ProjectID   string                `json:"project_id,omitempty" jsonschema:"Optional Multica project ID."`
	KnowledgeID string                `json:"knowledge_id,omitempty" jsonschema:"Optional published knowledge ID to revise."`
	Kind        knowledge.Kind        `json:"kind" jsonschema:"Knowledge kind."`
	Title       string                `json:"title" jsonschema:"Concise candidate title."`
	Content     string                `json:"content" jsonschema:"Proposed knowledge content."`
	Reason      string                `json:"reason" jsonschema:"Why this should become governed knowledge."`
	Sources     []knowledge.SourceRef `json:"source_refs,omitempty" jsonschema:"Optional provenance references."`
}

type mcpKnowledgePage struct {
	Entries             []map[string]any `json:"entries"`
	Candidates          []map[string]any `json:"candidates,omitempty"`
	NextCursor          string           `json:"next_cursor,omitempty"`
	CandidateNextCursor string           `json:"candidate_next_cursor,omitempty"`
}

type mcpKnowledgeEntry struct {
	Entry map[string]any `json:"entry"`
}

type mcpKnowledgeCandidate struct {
	Candidate map[string]any `json:"candidate"`
}

func (h *Handler) ConfigureKnowledgeMCP(
	publicURL string,
	authorizationServers []string,
	verifiers ...mcpauth.TokenVerifier,
) {
	h.knowledgeMCPPublicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	for _, server := range authorizationServers {
		server = strings.TrimRight(strings.TrimSpace(server), "/")
		if server != "" {
			h.knowledgeMCPAuthorizationServers = append(h.knowledgeMCPAuthorizationServers, server)
		}
	}
	if len(verifiers) > 0 {
		h.knowledgeMCPVerifier = verifiers[0]
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "multica-knowledge", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "knowledge_search", Description: "Search published workspace knowledge."}, h.mcpSearchKnowledge)
	mcp.AddTool(server, &mcp.Tool{Name: "knowledge_list", Description: "List published workspace knowledge."}, h.mcpListKnowledge)
	mcp.AddTool(server, &mcp.Tool{Name: "knowledge_get", Description: "Get governed knowledge with immutable revisions."}, h.mcpGetKnowledge)
	mcp.AddTool(server, &mcp.Tool{Name: "knowledge_propose", Description: "Create a candidate for human review; never publishes directly."}, h.mcpProposeKnowledge)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "knowledge://workspaces/{workspaceId}/entries/{knowledgeId}",
		Name:        "Multica knowledge entry", Description: "Governed workspace knowledge.",
		MIMEType: "application/json",
	}, h.mcpReadKnowledgeResource)

	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 1 << 20,
			PropagateRequestCancellation: true,
		},
	)
	protected := http.NewCrossOriginProtection().Handler(streamable)
	h.knowledgeMCPHandler = mcpauth.RequireBearerToken(
		h.verifyKnowledgeMCPToken,
		&mcpauth.RequireBearerTokenOptions{AllowMissingExpiration: false},
	)(protected)
}

func (h *Handler) KnowledgeMCPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.knowledgeMCPHandler == nil || h.knowledgeStore == nil {
			writeError(w, http.StatusServiceUnavailable, "knowledge store unavailable")
			return
		}
		if !h.validKnowledgeMCPOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid MCP origin")
			return
		}
		if _, _, ok := parseKnowledgeBearer(r.Header.Get("Authorization")); !ok {
			_, metadataURL, hasURL := h.knowledgeMCPResourceURLs(r, chi.URLParam(r, "workspaceSlug"))
			if hasURL {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(
					`Bearer resource_metadata=%q, scope=%q`,
					metadataURL, scopeKnowledgeRead+" "+scopeKnowledgePropose,
				))
			}
		}
		h.knowledgeMCPHandler.ServeHTTP(w, r)
	})
}

func (h *Handler) KnowledgeMCPMetadata(w http.ResponseWriter, r *http.Request) {
	resource, _, ok := h.knowledgeMCPResourceURLs(r, chi.URLParam(r, "workspaceSlug"))
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "MCP public URL is required for remote authorization")
		return
	}
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               resource,
		AuthorizationServers:   append([]string(nil), h.knowledgeMCPAuthorizationServers...),
		ScopesSupported:        []string{scopeKnowledgeRead, scopeKnowledgeCandidateRead, scopeKnowledgePropose},
		BearerMethodsSupported: []string{"header"}, ResourceName: "Multica Knowledge",
	}
	mcpauth.ProtectedResourceMetadataHandler(metadata).ServeHTTP(w, r)
}

func (h *Handler) verifyKnowledgeMCPToken(
	ctx context.Context,
	token string,
	request *http.Request,
) (*mcpauth.TokenInfo, error) {
	var tokenInfo *mcpauth.TokenInfo
	var err error
	if h.knowledgeMCPVerifier != nil {
		tokenInfo, err = h.knowledgeMCPVerifier(ctx, token, request)
		if err != nil {
			return nil, err
		}
		if tokenInfo == nil || strings.TrimSpace(tokenInfo.UserID) == "" {
			return nil, mcpauth.ErrInvalidToken
		}
	} else {
		if h.Queries == nil {
			return nil, mcpauth.ErrInvalidToken
		}
		pat, err := h.Queries.GetPersonalAccessTokenByHash(ctx, auth.HashToken(token))
		if err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		expiration := time.Now().UTC().Add(24 * time.Hour)
		if pat.ExpiresAt.Valid {
			expiration = pat.ExpiresAt.Time
		}
		tokenInfo = &mcpauth.TokenInfo{UserID: util.UUIDToString(pat.UserID), Expiration: expiration}
	}

	workspace, err := h.Queries.GetWorkspaceBySlug(ctx, strings.TrimSpace(chi.URLParam(request, "workspaceSlug")))
	if err != nil {
		return nil, mcpauth.ErrInvalidToken
	}
	userUUID, err := util.ParseUUID(tokenInfo.UserID)
	if err != nil {
		return nil, mcpauth.ErrInvalidToken
	}
	member, err := h.Queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
		UserID: userUUID, WorkspaceID: workspace.ID,
	})
	if err != nil {
		return nil, mcpauth.ErrInvalidToken
	}
	global, projectIDs, err := h.knowledgeMCPReviewScope(ctx, workspace.ID, member)
	if err != nil {
		return nil, err
	}
	allowed := []string{scopeKnowledgeRead, scopeKnowledgePropose}
	if global || len(projectIDs) > 0 {
		allowed = append(allowed, scopeKnowledgeCandidateRead)
	}
	if h.knowledgeMCPVerifier == nil {
		tokenInfo.Scopes = allowed
	} else {
		tokenInfo.Scopes = intersectKnowledgeScopes(tokenInfo.Scopes, allowed)
	}
	tokenInfo.Extra = map[string]any{
		"workspace_id":          util.UUIDToString(workspace.ID),
		"candidate_global":      global,
		"candidate_project_ids": projectIDs,
	}
	return tokenInfo, nil
}

func (h *Handler) knowledgeMCPReviewScope(
	ctx context.Context,
	workspaceID pgtype.UUID,
	member db.Member,
) (bool, []string, error) {
	if member.Role == "owner" || member.Role == "admin" {
		return true, nil, nil
	}
	projects, err := h.Queries.ListProjects(ctx, db.ListProjectsParams{WorkspaceID: workspaceID})
	if err != nil {
		return false, nil, err
	}
	projectIDs := make([]string, 0)
	for _, project := range projects {
		if project.LeadType.String == "member" && project.LeadID == member.UserID {
			projectIDs = append(projectIDs, util.UUIDToString(project.ID))
		}
	}
	return false, projectIDs, nil
}

func (h *Handler) mcpSearchKnowledge(ctx context.Context, _ *mcp.CallToolRequest, input mcpKnowledgeSearchInput) (*mcp.CallToolResult, mcpKnowledgePage, error) {
	return h.runMCPKnowledgeSearch(ctx, input, true)
}

func (h *Handler) mcpListKnowledge(ctx context.Context, _ *mcp.CallToolRequest, input mcpKnowledgeSearchInput) (*mcp.CallToolResult, mcpKnowledgePage, error) {
	input.Query = ""
	return h.runMCPKnowledgeSearch(ctx, input, false)
}

func (h *Handler) runMCPKnowledgeSearch(ctx context.Context, input mcpKnowledgeSearchInput, requireQuery bool) (*mcp.CallToolResult, mcpKnowledgePage, error) {
	identity, err := requireKnowledgeMCPIdentity(ctx, scopeKnowledgeRead)
	if err != nil {
		return nil, mcpKnowledgePage{}, err
	}
	if requireQuery && strings.TrimSpace(input.Query) == "" {
		return nil, mcpKnowledgePage{}, errors.New("query is required")
	}
	if input.ProjectID != "" && !h.knowledgeProjectInWorkspace(ctx, identity.WorkspaceID, input.ProjectID) {
		return nil, mcpKnowledgePage{}, errors.New("project not found in workspace")
	}
	query := knowledge.SearchQuery{
		WorkspaceID: identity.WorkspaceID, ProjectID: input.ProjectID,
		Text: input.Query, Limit: input.Limit, Cursor: input.Cursor,
	}
	if input.Kind != "" {
		query.Kinds = []knowledge.Kind{knowledge.Kind(input.Kind)}
	}
	page, err := h.knowledgeStore.Search(ctx, query)
	if err != nil {
		return nil, mcpKnowledgePage{}, err
	}
	result := mcpKnowledgePage{Entries: make([]map[string]any, 0, len(page.Results)), NextCursor: page.NextCursor}
	for _, item := range page.Results {
		entry := knowledgeEntryResponse(item.Entry)
		entry["citation"] = item.Citation
		entry["score"] = item.Score
		result.Entries = append(result.Entries, entry)
	}
	if input.IncludeCandidates {
		if !slices.Contains(identity.Scopes, scopeKnowledgeCandidateRead) {
			return nil, mcpKnowledgePage{}, errors.New("insufficient knowledge candidate scope")
		}
		if input.ProjectID != "" &&
			!identity.CandidateGlobal &&
			!slices.Contains(identity.CandidateProjectIDs, input.ProjectID) {
			return nil, mcpKnowledgePage{}, errors.New("project is outside the knowledge candidate scope")
		}
		candidateQuery := knowledge.CandidateQuery{
			WorkspaceID: identity.WorkspaceID, ProjectID: input.ProjectID,
			Limit: input.Limit, Cursor: input.CandidateCursor,
			Statuses: []knowledge.Status{knowledge.StatusCandidate, knowledge.StatusInReview},
		}
		if !identity.CandidateGlobal {
			candidateQuery.ProjectIDs = identity.CandidateProjectIDs
		}
		if input.Kind != "" {
			candidateQuery.Kinds = []knowledge.Kind{knowledge.Kind(input.Kind)}
		}
		candidates, err := h.knowledgeStore.ListCandidates(ctx, candidateQuery)
		if err != nil {
			return nil, mcpKnowledgePage{}, err
		}
		textQuery := strings.ToLower(strings.TrimSpace(input.Query))
		for _, candidate := range candidates.Candidates {
			if textQuery == "" || strings.Contains(strings.ToLower(candidate.Title+"\n"+candidate.Content), textQuery) {
				result.Candidates = append(result.Candidates, knowledgeCandidateResponse(candidate))
			}
		}
		result.CandidateNextCursor = candidates.NextCursor
	}
	return nil, result, nil
}

func (h *Handler) mcpGetKnowledge(ctx context.Context, _ *mcp.CallToolRequest, input mcpKnowledgeGetInput) (*mcp.CallToolResult, mcpKnowledgeEntry, error) {
	identity, err := requireKnowledgeMCPIdentity(ctx, scopeKnowledgeRead)
	if err != nil {
		return nil, mcpKnowledgeEntry{}, err
	}
	entry, err := h.knowledgeStore.GetEntry(ctx, identity.WorkspaceID, input.KnowledgeID)
	if err != nil {
		return nil, mcpKnowledgeEntry{}, err
	}
	return nil, mcpKnowledgeEntry{Entry: knowledgeEntryResponse(entry)}, nil
}

func (h *Handler) mcpProposeKnowledge(ctx context.Context, _ *mcp.CallToolRequest, input mcpKnowledgeProposeInput) (*mcp.CallToolResult, mcpKnowledgeCandidate, error) {
	identity, err := requireKnowledgeMCPIdentity(ctx, scopeKnowledgePropose)
	if err != nil {
		return nil, mcpKnowledgeCandidate{}, err
	}
	candidate, err := h.knowledgeService.Propose(ctx, knowledge.ProposalInput{
		WorkspaceID: identity.WorkspaceID, ProjectID: input.ProjectID,
		TargetEntryID: input.KnowledgeID, Kind: input.Kind, Title: input.Title,
		Content: input.Content, Reason: input.Reason, ProposedBy: identity.UserID,
		SourceRefs: input.Sources,
	})
	if err != nil {
		return nil, mcpKnowledgeCandidate{}, err
	}
	return nil, mcpKnowledgeCandidate{Candidate: knowledgeCandidateResponse(candidate)}, nil
}

func (h *Handler) mcpReadKnowledgeResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	identity, err := requireKnowledgeMCPIdentity(ctx, scopeKnowledgeRead)
	if err != nil {
		return nil, err
	}
	workspaceID, knowledgeID, ok := parseKnowledgeResourceURI(request.Params.URI)
	if !ok || workspaceID != identity.WorkspaceID {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	entry, err := h.knowledgeStore.GetEntry(ctx, workspaceID, knowledgeID)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	payload, err := json.Marshal(knowledgeEntryResponse(entry))
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
		URI: request.Params.URI, MIMEType: "application/json", Text: string(payload),
	}}}, nil
}

type knowledgeMCPIdentity struct {
	UserID              string
	WorkspaceID         string
	Scopes              []string
	CandidateGlobal     bool
	CandidateProjectIDs []string
}

func requireKnowledgeMCPIdentity(ctx context.Context, scope string) (knowledgeMCPIdentity, error) {
	tokenInfo := mcpauth.TokenInfoFromContext(ctx)
	if tokenInfo == nil || !slices.Contains(tokenInfo.Scopes, scope) {
		return knowledgeMCPIdentity{}, errors.New("insufficient knowledge scope")
	}
	workspaceID, _ := tokenInfo.Extra["workspace_id"].(string)
	global, _ := tokenInfo.Extra["candidate_global"].(bool)
	projectIDs, _ := tokenInfo.Extra["candidate_project_ids"].([]string)
	if tokenInfo.UserID == "" || workspaceID == "" {
		return knowledgeMCPIdentity{}, errors.New("invalid MCP identity")
	}
	return knowledgeMCPIdentity{
		UserID: tokenInfo.UserID, WorkspaceID: workspaceID,
		Scopes:          append([]string(nil), tokenInfo.Scopes...),
		CandidateGlobal: global, CandidateProjectIDs: append([]string(nil), projectIDs...),
	}, nil
}

func intersectKnowledgeScopes(actual, allowed []string) []string {
	result := make([]string, 0, len(actual))
	for _, scope := range actual {
		if slices.Contains(allowed, scope) {
			result = append(result, scope)
		}
	}
	return result
}

func parseKnowledgeResourceURI(raw string) (string, string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "knowledge" || parsed.Host != "workspaces" {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 3 || parts[1] != "entries" {
		return "", "", false
	}
	return parts[0], parts[2], true
}

func (h *Handler) knowledgeMCPResourceURLs(request *http.Request, workspaceSlug string) (string, string, bool) {
	base := h.knowledgeMCPPublicURL
	if base == "" && isKnowledgeLocalHost(request.Host) {
		base = "http://" + request.Host
	}
	if base == "" {
		return "", "", false
	}
	if workspaceSlug == "" {
		return base + "/mcp", base + "/.well-known/oauth-protected-resource", true
	}
	resource := base + "/mcp/" + url.PathEscape(workspaceSlug) + "/knowledge"
	metadata := base + "/.well-known/oauth-protected-resource/mcp/" + url.PathEscape(workspaceSlug) + "/knowledge"
	return resource, metadata, true
}

func isKnowledgeLocalHost(host string) bool {
	hostname := host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		hostname = parsed
	}
	hostname = strings.Trim(strings.ToLower(hostname), "[]")
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

func (h *Handler) validKnowledgeMCPOrigin(request *http.Request) bool {
	rawOrigin := strings.TrimSpace(request.Header.Get("Origin"))
	if rawOrigin == "" {
		return true
	}
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Host == "" {
		return false
	}
	if strings.EqualFold(origin.Host, request.Host) {
		return true
	}
	if h.knowledgeMCPPublicURL != "" {
		publicURL, err := url.Parse(h.knowledgeMCPPublicURL)
		return err == nil && strings.EqualFold(origin.Host, publicURL.Host)
	}
	return false
}

func parseKnowledgeBearer(value string) (string, string, bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
		return "", "", false
	}
	return fields[0], fields[1], true
}
