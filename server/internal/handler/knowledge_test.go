package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/memory"
	"github.com/multica-ai/multica/server/internal/middleware"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestKnowledgeRoutesServeConfiguredStore(t *testing.T) {
	store := memory.New()
	service := knowledge.NewService(store, nil)
	ctx := context.Background()
	candidate, err := service.Propose(ctx, knowledge.ProposalInput{
		WorkspaceID: "11111111-1111-1111-1111-111111111111",
		Kind:        knowledge.KindLesson,
		Title:       "Retain delivery evidence",
		Content:     "Retry knowledge delivery without deleting source evidence.",
		Reason:      "Captured during recovery testing.",
		ProposedBy:  "22222222-2222-2222-2222-222222222222",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Review(ctx, knowledge.ReviewInput{
		WorkspaceID:      candidate.WorkspaceID,
		CandidateID:      candidate.ID,
		ExpectedRevision: 1,
		Action:           knowledge.ReviewApprove,
		ReviewerID:       "22222222-2222-2222-2222-222222222222",
		Rationale:        "Verified by the adapter recovery test.",
	}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{}
	h.ConfigureKnowledge(store, service, nil)
	router := chi.NewRouter()
	router.Route("/api/knowledge", h.RegisterKnowledgeRoutes)

	response := serveKnowledgeRequest(t, router, http.MethodGet, "/api/knowledge", nil, "owner")
	if response.Code != http.StatusOK {
		t.Fatalf("list knowledge status = %d body=%s", response.Code, response.Body.String())
	}
	var page map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page["total"] != float64(1) {
		t.Fatalf("knowledge page = %#v", page)
	}
}

func TestKnowledgeRoutesKeepCandidateGovernanceHiddenFromMembers(t *testing.T) {
	store := memory.New()
	service := knowledge.NewService(store, nil)
	h := &Handler{}
	h.ConfigureKnowledge(store, service, nil)
	router := chi.NewRouter()
	router.Route("/api/knowledge", h.RegisterKnowledgeRoutes)

	proposal := serveKnowledgeRequest(t, router, http.MethodPost, "/api/knowledge/proposals", map[string]any{
		"kind": "lesson", "title": "Recovery lesson",
		"content": "Retain source evidence.", "reason": "Observed during delivery.",
	}, "member")
	if proposal.Code != http.StatusCreated {
		t.Fatalf("proposal status = %d body=%s", proposal.Code, proposal.Body.String())
	}
	candidates := serveKnowledgeRequest(t, router, http.MethodGet, "/api/knowledge/candidates", nil, "member")
	if candidates.Code != http.StatusForbidden {
		t.Fatalf("candidate list status = %d body=%s", candidates.Code, candidates.Body.String())
	}
}

func TestKnowledgeMCPPublishesDiscoveryAndRejectsUnsafeRequests(t *testing.T) {
	store := memory.New()
	h := &Handler{}
	h.ConfigureKnowledge(store, knowledge.NewService(store, nil), nil)
	h.ConfigureKnowledgeMCP("http://localhost:8080", []string{"https://auth.example"})
	router := chi.NewRouter()
	router.Get("/.well-known/oauth-protected-resource/mcp/{workspaceSlug}/knowledge", h.KnowledgeMCPMetadata)
	router.Handle("/mcp/{workspaceSlug}/knowledge", h.KnowledgeMCPHandler())

	metadataRequest := httptest.NewRequest(
		http.MethodGet,
		"/.well-known/oauth-protected-resource/mcp/acme/knowledge",
		nil,
	)
	metadata := httptest.NewRecorder()
	router.ServeHTTP(metadata, metadataRequest)
	if metadata.Code != http.StatusOK || !strings.Contains(metadata.Body.String(), "knowledge:read") {
		t.Fatalf("protected resource metadata: status=%d body=%s", metadata.Code, metadata.Body.String())
	}

	missingToken := httptest.NewRequest(http.MethodPost, "/mcp/acme/knowledge", bytes.NewBufferString(`{}`))
	missingToken.Host = "localhost:8080"
	missingResponse := httptest.NewRecorder()
	router.ServeHTTP(missingResponse, missingToken)
	if missingResponse.Code != http.StatusUnauthorized || missingResponse.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("missing token response: status=%d headers=%v", missingResponse.Code, missingResponse.Header())
	}

	unsafeOrigin := httptest.NewRequest(http.MethodPost, "/mcp/acme/knowledge", bytes.NewBufferString(`{}`))
	unsafeOrigin.Header.Set("Origin", "https://attacker.example")
	unsafeResponse := httptest.NewRecorder()
	router.ServeHTTP(unsafeResponse, unsafeOrigin)
	if unsafeResponse.Code != http.StatusForbidden {
		t.Fatalf("unsafe origin status = %d, want %d", unsafeResponse.Code, http.StatusForbidden)
	}
}

func serveKnowledgeRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body any,
	role string,
) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	workspaceID := "11111111-1111-1111-1111-111111111111"
	member := db.Member{
		UserID:      pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
		WorkspaceID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		Role:        role,
	}
	request = request.WithContext(middleware.SetMemberContext(request.Context(), workspaceID, member))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
