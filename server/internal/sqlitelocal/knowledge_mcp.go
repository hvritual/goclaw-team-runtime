package sqlitelocal

import (
	"context"
	"database/sql"
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
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

const (
	scopeKnowledgeRead          = "knowledge:read"
	scopeKnowledgeCandidateRead = "knowledge:candidate:read"
	scopeKnowledgePropose       = "knowledge:propose"
)

type mcpSearchInput struct {
	Query             string `json:"query,omitempty" jsonschema:"Text to search for in published knowledge."`
	ProjectID         string `json:"project_id,omitempty" jsonschema:"Optional Multica project ID."`
	Kind              string `json:"kind,omitempty" jsonschema:"Optional knowledge kind filter."`
	IncludeCandidates bool   `json:"include_candidates,omitempty" jsonschema:"Include governed candidates when the token has knowledge:candidate:read."`
	Limit             int    `json:"limit,omitempty" jsonschema:"Maximum number of results, from 1 to 100."`
	Cursor            string `json:"cursor,omitempty" jsonschema:"Opaque pagination cursor from a previous result."`
}

type mcpGetInput struct {
	KnowledgeID string `json:"knowledge_id" jsonschema:"Published knowledge entry ID."`
}

type mcpProposeInput struct {
	ProjectID string                `json:"project_id,omitempty" jsonschema:"Optional Multica project ID."`
	Kind      knowledge.Kind        `json:"kind" jsonschema:"Knowledge kind."`
	Title     string                `json:"title" jsonschema:"Concise candidate title."`
	Content   string                `json:"content" jsonschema:"Proposed knowledge content."`
	Reason    string                `json:"reason" jsonschema:"Why this content should become governed knowledge."`
	Sources   []knowledge.SourceRef `json:"source_refs,omitempty" jsonschema:"Optional provenance references."`
}

type mcpKnowledgePage struct {
	Entries    []map[string]any `json:"entries"`
	Candidates []map[string]any `json:"candidates,omitempty"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type mcpKnowledgeEntry struct {
	Entry map[string]any `json:"entry"`
}

type mcpKnowledgeCandidate struct {
	Candidate map[string]any `json:"candidate"`
}

func (s *Server) configureKnowledgeMCP(options Options) {
	s.mcpPublicURL = strings.TrimRight(strings.TrimSpace(options.PublicURL), "/")
	for _, server := range options.MCPAuthorizationServers {
		server = strings.TrimRight(strings.TrimSpace(server), "/")
		if server != "" {
			s.mcpAuthorizationServers = append(s.mcpAuthorizationServers, server)
		}
	}
	s.mcpExternalVerifier = options.MCPTokenVerifier

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "multica-knowledge",
		Version: "1.0.0",
	}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "knowledge_search",
		Description: "Search published knowledge and, with explicit scope, governed candidates in the authenticated Multica workspace.",
	}, s.mcpSearchKnowledge)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "knowledge_list",
		Description: "List published knowledge and, with explicit scope, governed candidates in the authenticated Multica workspace.",
	}, s.mcpListKnowledge)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "knowledge_get",
		Description: "Get a published knowledge entry and its governed revisions.",
	}, s.mcpGetKnowledge)
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "knowledge_propose",
		Description: "Create a candidate for human review; this never publishes directly.",
	}, s.mcpProposeKnowledge)

	mcpServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "knowledge://workspaces/{workspaceId}/entries/{knowledgeId}",
		Name:        "Multica knowledge entry",
		Description: "A governed knowledge entry in a Multica workspace.",
		MIMEType:    "application/json",
	}, s.mcpReadKnowledgeResource)

	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{
			Stateless:                    true,
			JSONResponse:                 true,
			MaxRequestBodyBytes:          1 << 20,
			PropagateRequestCancellation: true,
		},
	)
	protection := http.NewCrossOriginProtection()
	protected := protection.Handler(streamable)
	s.mcpHandler = mcpauth.RequireBearerToken(
		s.verifyMCPToken,
		&mcpauth.RequireBearerTokenOptions{
			AllowMissingExpiration: false,
		},
	)(protected)
}

func (s *Server) handleKnowledgeMCP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.mcpHandler == nil {
			writeError(w, http.StatusServiceUnavailable, errKnowledgeDisabled.Error())
			return
		}
		if !s.validMCPOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid MCP origin")
			return
		}
		if _, _, ok := parseBearerHeader(r.Header.Get("Authorization")); !ok {
			workspaceSlug := strings.TrimSpace(chi.URLParam(r, "workspaceSlug"))
			_, metadataURL, hasURL := s.mcpResourceURLs(r, workspaceSlug)
			if hasURL {
				w.Header().Set(
					"WWW-Authenticate",
					fmt.Sprintf(
						`Bearer resource_metadata=%q, scope=%q`,
						metadataURL,
						scopeKnowledgeRead+" "+scopeKnowledgePropose,
					),
				)
			}
		}
		s.mcpHandler.ServeHTTP(w, r)
	})
}

func (s *Server) verifyMCPToken(
	ctx context.Context,
	token string,
	request *http.Request,
) (*mcpauth.TokenInfo, error) {
	var tokenInfo *mcpauth.TokenInfo
	var err error
	if s.mcpExternalVerifier != nil {
		tokenInfo, err = s.mcpExternalVerifier(ctx, token, request)
		if err != nil {
			return nil, err
		}
		if tokenInfo == nil || strings.TrimSpace(tokenInfo.UserID) == "" {
			return nil, fmt.Errorf("%w: token has no user identity", mcpauth.ErrInvalidToken)
		}
	} else {
		var createdAt string
		tokenInfo = &mcpauth.TokenInfo{}
		err = s.db.QueryRowContext(ctx, `
			SELECT user_id, created_at
			FROM auth_tokens
			WHERE token = ?`,
			token,
		).Scan(&tokenInfo.UserID, &createdAt)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, mcpauth.ErrInvalidToken
		}
		if err != nil {
			return nil, err
		}
		created, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, mcpauth.ErrInvalidToken
		}
		tokenInfo.Expiration = created.Add(7 * 24 * time.Hour)
	}

	workspaceSlug := strings.TrimSpace(chi.URLParam(request, "workspaceSlug"))
	if workspaceSlug == "" {
		workspaceSlug = workspaceSlugFromMCPPath(request.URL.Path)
	}
	var workspaceID, role string
	err = s.db.QueryRowContext(ctx, `
		SELECT w.id, m.role
		FROM workspaces w
		JOIN members m ON m.workspace_id = w.id
		WHERE w.slug = ? AND m.user_id = ?`,
		workspaceSlug,
		tokenInfo.UserID,
	).Scan(&workspaceID, &role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, mcpauth.ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	allowedScopes := []string{scopeKnowledgeRead, scopeKnowledgePropose}
	if role == string(workspacepermissions.RoleOwner) || role == string(workspacepermissions.RoleAdmin) {
		allowedScopes = append(allowedScopes, scopeKnowledgeCandidateRead)
	}
	if s.mcpExternalVerifier == nil {
		tokenInfo.Scopes = append([]string(nil), allowedScopes...)
	} else {
		tokenInfo.Scopes = intersectScopes(tokenInfo.Scopes, allowedScopes)
	}
	tokenInfo.Extra = map[string]any{
		"workspace_id":   workspaceID,
		"workspace_slug": workspaceSlug,
		"role":           role,
	}
	return tokenInfo, nil
}

func (s *Server) getMCPProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	workspaceSlug := strings.TrimSpace(chi.URLParam(r, "workspaceSlug"))
	resource, metadataURL, ok := s.mcpResourceURLs(r, workspaceSlug)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "MCP public URL is required for remote authorization")
		return
	}
	_ = metadataURL
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:             resource,
		AuthorizationServers: append([]string(nil), s.mcpAuthorizationServers...),
		ScopesSupported: []string{
			scopeKnowledgeRead,
			scopeKnowledgeCandidateRead,
			scopeKnowledgePropose,
		},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Multica Knowledge",
	}
	mcpauth.ProtectedResourceMetadataHandler(metadata).ServeHTTP(w, r)
}

func (s *Server) mcpSearchKnowledge(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input mcpSearchInput,
) (*mcp.CallToolResult, mcpKnowledgePage, error) {
	return s.runMCPSearch(ctx, input, true)
}

func (s *Server) mcpListKnowledge(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input mcpSearchInput,
) (*mcp.CallToolResult, mcpKnowledgePage, error) {
	input.Query = ""
	return s.runMCPSearch(ctx, input, false)
}

func (s *Server) runMCPSearch(
	ctx context.Context,
	input mcpSearchInput,
	requireQuery bool,
) (*mcp.CallToolResult, mcpKnowledgePage, error) {
	identity, err := requireMCPIdentity(ctx, scopeKnowledgeRead)
	if err != nil {
		return nil, mcpKnowledgePage{}, err
	}
	if requireQuery && strings.TrimSpace(input.Query) == "" {
		return nil, mcpKnowledgePage{}, errors.New("query is required")
	}
	if input.ProjectID != "" &&
		!s.belongsToWorkspace(ctx, "projects", input.ProjectID, identity.WorkspaceID) {
		return nil, mcpKnowledgePage{}, errors.New("project not found in workspace")
	}
	query := knowledge.SearchQuery{
		WorkspaceID: identity.WorkspaceID,
		ProjectID:   input.ProjectID,
		Text:        input.Query,
		Limit:       input.Limit,
		Cursor:      input.Cursor,
	}
	if input.Kind != "" {
		query.Kinds = []knowledge.Kind{knowledge.Kind(input.Kind)}
	}
	page, err := s.knowledgeStore.Search(ctx, query)
	if err != nil {
		return nil, mcpKnowledgePage{}, err
	}
	entries := make([]map[string]any, 0, len(page.Results))
	for _, result := range page.Results {
		entry := knowledgeEntryResponse(result.Entry)
		entry["citation"] = result.Citation
		entry["score"] = result.Score
		entries = append(entries, entry)
	}
	result := mcpKnowledgePage{Entries: entries, NextCursor: page.NextCursor}
	if input.IncludeCandidates {
		if !slices.Contains(identity.Scopes, scopeKnowledgeCandidateRead) {
			return nil, mcpKnowledgePage{}, errors.New("insufficient knowledge candidate scope")
		}
		candidateQuery := knowledge.CandidateQuery{
			WorkspaceID: identity.WorkspaceID,
			ProjectID:   input.ProjectID,
			Limit:       input.Limit,
			Cursor:      input.Cursor,
		}
		if input.Kind != "" {
			candidateQuery.Kinds = []knowledge.Kind{knowledge.Kind(input.Kind)}
		}
		candidates, err := s.knowledgeStore.ListCandidates(ctx, candidateQuery)
		if err != nil {
			return nil, mcpKnowledgePage{}, err
		}
		textQuery := strings.ToLower(strings.TrimSpace(input.Query))
		for _, candidate := range candidates.Candidates {
			haystack := strings.ToLower(candidate.Title + "\n" + candidate.Content)
			if textQuery != "" && !strings.Contains(haystack, textQuery) {
				continue
			}
			result.Candidates = append(result.Candidates, knowledgeCandidateResponse(candidate))
		}
	}
	return nil, result, nil
}

func (s *Server) mcpGetKnowledge(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input mcpGetInput,
) (*mcp.CallToolResult, mcpKnowledgeEntry, error) {
	identity, err := requireMCPIdentity(ctx, scopeKnowledgeRead)
	if err != nil {
		return nil, mcpKnowledgeEntry{}, err
	}
	entry, err := s.knowledgeStore.GetEntry(ctx, identity.WorkspaceID, input.KnowledgeID)
	if err != nil {
		return nil, mcpKnowledgeEntry{}, err
	}
	return nil, mcpKnowledgeEntry{Entry: knowledgeEntryResponse(entry)}, nil
}

func (s *Server) mcpProposeKnowledge(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input mcpProposeInput,
) (*mcp.CallToolResult, mcpKnowledgeCandidate, error) {
	identity, err := requireMCPIdentity(ctx, scopeKnowledgePropose)
	if err != nil {
		return nil, mcpKnowledgeCandidate{}, err
	}
	if input.ProjectID != "" &&
		!s.belongsToWorkspace(ctx, "projects", input.ProjectID, identity.WorkspaceID) {
		return nil, mcpKnowledgeCandidate{}, errors.New("project not found in workspace")
	}
	candidate, err := s.knowledgeService.Propose(ctx, knowledge.ProposalInput{
		WorkspaceID: identity.WorkspaceID,
		ProjectID:   input.ProjectID,
		Kind:        input.Kind,
		Title:       input.Title,
		Content:     input.Content,
		Reason:      input.Reason,
		ProposedBy:  identity.UserID,
		SourceRefs:  input.Sources,
	})
	if err != nil {
		return nil, mcpKnowledgeCandidate{}, err
	}
	return nil, mcpKnowledgeCandidate{Candidate: knowledgeCandidateResponse(candidate)}, nil
}

func (s *Server) mcpReadKnowledgeResource(
	ctx context.Context,
	request *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	identity, err := requireMCPIdentity(ctx, scopeKnowledgeRead)
	if err != nil {
		return nil, err
	}
	workspaceID, knowledgeID, ok := parseKnowledgeResourceURI(request.Params.URI)
	if !ok || workspaceID != identity.WorkspaceID {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	entry, err := s.knowledgeStore.GetEntry(ctx, workspaceID, knowledgeID)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	payload, err := json.Marshal(knowledgeEntryResponse(entry))
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(payload),
		}},
	}, nil
}

type mcpIdentity struct {
	UserID      string
	WorkspaceID string
	Role        string
	Scopes      []string
}

func requireMCPIdentity(ctx context.Context, scope string) (mcpIdentity, error) {
	tokenInfo := mcpauth.TokenInfoFromContext(ctx)
	if tokenInfo == nil || !slices.Contains(tokenInfo.Scopes, scope) {
		return mcpIdentity{}, errors.New("insufficient knowledge scope")
	}
	workspaceID, _ := tokenInfo.Extra["workspace_id"].(string)
	role, _ := tokenInfo.Extra["role"].(string)
	if tokenInfo.UserID == "" || workspaceID == "" {
		return mcpIdentity{}, errors.New("invalid MCP identity")
	}
	return mcpIdentity{
		UserID:      tokenInfo.UserID,
		WorkspaceID: workspaceID,
		Role:        role,
		Scopes:      append([]string(nil), tokenInfo.Scopes...),
	}, nil
}

func intersectScopes(actual, allowed []string) []string {
	result := make([]string, 0, len(actual))
	for _, scope := range actual {
		if slices.Contains(allowed, scope) {
			result = append(result, scope)
		}
	}
	return result
}

func workspaceSlugFromMCPPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 3 && parts[0] == "mcp" && parts[2] == "knowledge" {
		value, _ := url.PathUnescape(parts[1])
		return value
	}
	return ""
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

func (s *Server) mcpResourceURLs(
	request *http.Request,
	workspaceSlug string,
) (resource string, metadata string, ok bool) {
	base := s.mcpPublicURL
	if base == "" && isLocalHost(request.Host) {
		base = "http://" + request.Host
	}
	if base == "" {
		return "", "", false
	}
	if workspaceSlug == "" {
		return base + "/mcp", base + "/.well-known/oauth-protected-resource", true
	}
	resource = base + "/mcp/" + url.PathEscape(workspaceSlug) + "/knowledge"
	metadata = base + "/.well-known/oauth-protected-resource/mcp/" +
		url.PathEscape(workspaceSlug) + "/knowledge"
	return resource, metadata, true
}

func isLocalHost(host string) bool {
	hostname := host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		hostname = parsed
	}
	hostname = strings.Trim(strings.ToLower(hostname), "[]")
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

func (s *Server) validMCPOrigin(request *http.Request) bool {
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
	if s.mcpPublicURL != "" {
		publicURL, err := url.Parse(s.mcpPublicURL)
		return err == nil && strings.EqualFold(origin.Host, publicURL.Host)
	}
	return false
}

func parseBearerHeader(value string) (scheme string, token string, ok bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
		return "", "", false
	}
	return fields[0], fields[1], true
}
