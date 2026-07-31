package handler

import (
	"context"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/auth"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestKnowledgeMCPAcceptsProjectLeadPATWithCandidateScope(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("integration test requires Postgres at DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	userID := uuid.NewString()
	workspaceID := uuid.NewString()
	projectID := uuid.NewString()
	slug := "knowledge-mcp-" + uuid.NewString()
	token := "mul_" + uuid.NewString()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO "user"(id, name, email) VALUES ($1, 'Knowledge Lead', $2)`, []any{userID, slug + "@example.test"}},
		{`INSERT INTO workspace(id, name, slug, issue_prefix) VALUES ($1, 'Knowledge Workspace', $2, 'KNW')`, []any{workspaceID, slug}},
		{`INSERT INTO member(workspace_id, user_id, role) VALUES ($1, $2, 'member')`, []any{workspaceID, userID}},
		{`INSERT INTO personal_access_token(user_id, name, token_hash, token_prefix) VALUES ($1, 'Knowledge MCP', $2, 'mul_test')`, []any{userID, auth.HashToken(token)}},
		{`INSERT INTO project(id, workspace_id, title, lead_type, lead_id) VALUES ($1, $2, 'Governed delivery', 'member', $3)`, []any{projectID, workspaceID, userID}},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}

	h := &Handler{Queries: db.New(tx)}
	request := httptest.NewRequest("POST", "/mcp/"+slug+"/knowledge", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("workspaceSlug", slug)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	info, err := h.verifyKnowledgeMCPToken(request.Context(), token, request)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(info.Scopes, scopeKnowledgeCandidateRead) {
		t.Fatalf("scopes = %#v", info.Scopes)
	}
	projectIDs, ok := info.Extra["candidate_project_ids"].([]string)
	if !ok || !slices.Contains(projectIDs, projectID) {
		t.Fatalf("token extra = %#v", info.Extra)
	}
}
