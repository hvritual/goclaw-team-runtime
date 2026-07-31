package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/middleware"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestProposeCommentDecisionIsAuthorizedRevisionedAndIdempotent(t *testing.T) {
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
	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer outer.Rollback(ctx)

	userID := uuid.NewString()
	workspaceID := uuid.NewString()
	otherWorkspaceID := uuid.NewString()
	projectID := uuid.NewString()
	issueID := uuid.NewString()
	commentID := uuid.NewString()
	slug := "comment-knowledge-" + uuid.NewString()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO "user"(id, name, email) VALUES ($1, 'Comment Member', $2)`, []any{userID, slug + "@example.test"}},
		{`INSERT INTO workspace(id, name, slug, issue_prefix) VALUES ($1, 'Knowledge Workspace', $2, 'KNW')`, []any{workspaceID, slug}},
		{`INSERT INTO workspace(id, name, slug, issue_prefix) VALUES ($1, 'Other Workspace', $2, 'OTH')`, []any{otherWorkspaceID, slug + "-other"}},
		{`INSERT INTO member(workspace_id, user_id, role) VALUES ($1, $2, 'member')`, []any{workspaceID, userID}},
		{`INSERT INTO project(id, workspace_id, title) VALUES ($1, $2, 'Knowledge delivery')`, []any{projectID, workspaceID}},
		{`INSERT INTO issue(id, workspace_id, project_id, title, creator_type, creator_id, number) VALUES ($1, $2, $3, 'Capture decisions', 'member', $4, 1)`, []any{issueID, workspaceID, projectID, userID}},
		{`INSERT INTO comment(id, issue_id, workspace_id, author_type, author_id, content) VALUES ($1, $2, $3, 'member', $4, 'Use SQLite first.')`, []any{commentID, issueID, workspaceID, userID}},
	}
	for _, statement := range statements {
		if _, err := outer.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}

	h := &Handler{Queries: db.New(outer), TxStarter: outer, knowledgeEvidenceEnabled: true}
	router := chi.NewRouter()
	router.Post("/api/comments/{commentId}/knowledge-proposals", h.ProposeCommentDecision)
	member := db.Member{
		UserID:      parseUUID(userID),
		WorkspaceID: parseUUID(workspaceID),
		Role:        "member",
	}

	first := serveCommentKnowledgeRequest(t, router, commentID, userID, workspaceID, member)
	if first.Code != http.StatusAccepted {
		t.Fatalf("first capture status = %d body=%s", first.Code, first.Body.String())
	}
	var firstBody struct {
		Queued         bool   `json:"queued"`
		EvidenceID     string `json:"evidence_id"`
		SourceRevision string `json:"source_revision"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}
	if !firstBody.Queued || firstBody.EvidenceID == "" || firstBody.SourceRevision == "" {
		t.Fatalf("first capture = %#v", firstBody)
	}

	repeat := serveCommentKnowledgeRequest(t, router, commentID, userID, workspaceID, member)
	if repeat.Code != http.StatusOK {
		t.Fatalf("repeat capture status = %d body=%s", repeat.Code, repeat.Body.String())
	}
	var repeatBody struct {
		Queued bool `json:"queued"`
	}
	if err := json.Unmarshal(repeat.Body.Bytes(), &repeatBody); err != nil {
		t.Fatal(err)
	}
	if repeatBody.Queued {
		t.Fatal("same comment revision was queued twice")
	}

	if _, err := outer.Exec(ctx, `UPDATE comment SET content = 'Use SQLite first, behind a port.', updated_at = updated_at + interval '1 second' WHERE id = $1`, commentID); err != nil {
		t.Fatal(err)
	}
	edited := serveCommentKnowledgeRequest(t, router, commentID, userID, workspaceID, member)
	if edited.Code != http.StatusAccepted {
		t.Fatalf("edited capture status = %d body=%s", edited.Code, edited.Body.String())
	}

	var count, distinctRevisions int
	if err := outer.QueryRow(ctx, `
		SELECT count(*), count(DISTINCT payload_json->>'SourceRevision')
		FROM knowledge_evidence_outbox
		WHERE workspace_id = $1
		  AND payload_json->>'EventType' = 'comment.decision_proposed'`, workspaceID).Scan(&count, &distinctRevisions); err != nil {
		t.Fatal(err)
	}
	if count != 2 || distinctRevisions != 2 {
		t.Fatalf("comment decision outbox rows = %d revisions = %d, want 2 and 2", count, distinctRevisions)
	}

	crossWorkspace := serveCommentKnowledgeRequest(t, router, commentID, userID, otherWorkspaceID, member)
	if crossWorkspace.Code != http.StatusNotFound {
		t.Fatalf("cross-workspace status = %d body=%s", crossWorkspace.Code, crossWorkspace.Body.String())
	}
}

func serveCommentKnowledgeRequest(
	t *testing.T,
	handler http.Handler,
	commentID string,
	userID string,
	workspaceID string,
	member db.Member,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/comments/"+commentID+"/knowledge-proposals", bytes.NewReader(nil))
	request.Header.Set("X-User-ID", userID)
	request = request.WithContext(middleware.SetMemberContext(request.Context(), workspaceID, member))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
