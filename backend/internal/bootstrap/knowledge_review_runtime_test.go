package bootstrap

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestInstalledKnowledgeReviewProposalIndependentPublishAndReadback(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "knowledge-review.db"), "knowledge-review", "knowledge-owner-review@example.com")
	member := addCollaborationRuntimeMember(t, fixture, "knowledge-proposer@example.com", "member")
	memberHeaders := collaborationHeaders(member.Token, fixture.workspaceSlug)
	memberHeaders["Idempotency-Key"] = "proposal-1"
	body := `{"kind":"lesson","title":"Retain accepted behavior","content":"Keep exact acceptance evidence.","reason":"Reusable recovery guidance","source_refs":[{"type":"acceptance_conclusion","id":"issue-1","revision":"sha256:abc","citation":"Acceptance passed","asset_id":null,"asset_version_id":null}]}`
	proposal := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/proposals", body, memberHeaders)
	var candidate struct {
		ID       string `json:"id"`
		Revision int    `json:"revision"`
	}
	if proposal.Code != http.StatusCreated || json.Unmarshal(proposal.Body.Bytes(), &candidate) != nil || candidate.ID == "" || candidate.Revision != 1 {
		t.Fatalf("proposal = %d %s", proposal.Code, proposal.Body.String())
	}
	replay := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/proposals", body, memberHeaders)
	if replay.Code != http.StatusCreated || !containsJSON(replay.Body.Bytes(), `"id":"`+candidate.ID+`"`) {
		t.Fatalf("proposal replay = %d %s", replay.Code, replay.Body.String())
	}
	memberQueue := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge/candidates?limit=50", "", memberHeaders)
	if memberQueue.Code != http.StatusForbidden {
		t.Fatalf("member queue = %d %s", memberQueue.Code, memberQueue.Body.String())
	}
	queue := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge/candidates?limit=50", "", fixture.headers)
	if queue.Code != http.StatusOK || !containsJSON(queue.Body.Bytes(), `"id":"`+candidate.ID+`"`) {
		t.Fatalf("owner queue = %d %s", queue.Code, queue.Body.String())
	}
	approve := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/candidates/"+candidate.ID+"/review", `{"action":"approve","expected_revision":1,"rationale":"Independent evidence review","emergency":false}`, fixture.headers)
	if approve.Code != http.StatusOK || !containsJSON(approve.Body.Bytes(), `"status":"in_review"`, `"revision":2`) {
		t.Fatalf("approve = %d %s", approve.Code, approve.Body.String())
	}
	stale := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/candidates/"+candidate.ID+"/review", `{"action":"publish","expected_revision":1,"rationale":"Stale publish","emergency":false}`, fixture.headers)
	if stale.Code != http.StatusConflict || !containsJSON(stale.Body.Bytes(), `"code":"revision_conflict"`) {
		t.Fatalf("stale = %d %s", stale.Code, stale.Body.String())
	}
	publish := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/candidates/"+candidate.ID+"/review", `{"action":"publish","expected_revision":2,"rationale":"Evidence is complete","emergency":false}`, fixture.headers)
	var published struct {
		Entry *struct {
			ID string `json:"id"`
		} `json:"entry"`
	}
	if publish.Code != http.StatusOK || json.Unmarshal(publish.Body.Bytes(), &published) != nil || published.Entry == nil || published.Entry.ID == "" {
		t.Fatalf("publish = %d %s", publish.Code, publish.Body.String())
	}
	readback := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge/"+published.Entry.ID, "", memberHeaders)
	if readback.Code != http.StatusOK || !containsJSON(readback.Body.Bytes(), `"title":"Retain accepted behavior"`, `"citation":"Acceptance passed"`) {
		t.Fatalf("readback = %d %s", readback.Code, readback.Body.String())
	}
	var candidateCount, auditCount int
	if err := fixture.runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_knowledge_candidates WHERE workspace_id=?`, fixture.workspaceID).Scan(&candidateCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_audit_entries WHERE workspace_id=? AND resource_kind='knowledge_candidate'`, fixture.workspaceID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if candidateCount != 1 || auditCount != 3 {
		t.Fatalf("counts candidate=%d audit=%d", candidateCount, auditCount)
	}
}

func TestInstalledKnowledgeReviewSelfReviewRequiresOwnerEmergencyOverride(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "knowledge-self-review.db"), "knowledge-self-review", "knowledge-owner-self@example.com")
	body := `{"kind":"lesson","title":"Emergency evidence","content":"Retain this evidence.","reason":"Urgent governed recovery","source_refs":[{"type":"acceptance_conclusion","id":"issue-2","revision":"sha256:def","citation":"Emergency acceptance","asset_id":null,"asset_version_id":null}]}`
	ownerHeaders := collaborationHeaders(fixture.login.Token, fixture.workspaceSlug)
	ownerHeaders["Idempotency-Key"] = "owner-proposal"
	proposal := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/proposals", body, ownerHeaders)
	var ownerCandidate struct {
		ID string `json:"id"`
	}
	if proposal.Code != http.StatusCreated || json.Unmarshal(proposal.Body.Bytes(), &ownerCandidate) != nil {
		t.Fatalf("owner proposal = %d %s", proposal.Code, proposal.Body.String())
	}
	denied := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/candidates/"+ownerCandidate.ID+"/review", `{"action":"approve","expected_revision":1,"rationale":"ordinary self review","emergency":false}`, ownerHeaders)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("owner ordinary self review = %d %s", denied.Code, denied.Body.String())
	}
	override := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/candidates/"+ownerCandidate.ID+"/review", `{"action":"approve","expected_revision":1,"rationale":"documented emergency override","emergency":true}`, ownerHeaders)
	if override.Code != http.StatusOK || !containsJSON(override.Body.Bytes(), `"status":"in_review"`) {
		t.Fatalf("owner emergency = %d %s", override.Code, override.Body.String())
	}
	admin := addCollaborationRuntimeMember(t, fixture, "knowledge-admin-self@example.com", "admin")
	adminHeaders := collaborationHeaders(admin.Token, fixture.workspaceSlug)
	adminHeaders["Idempotency-Key"] = "admin-proposal"
	adminProposal := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/proposals", body, adminHeaders)
	var adminCandidate struct {
		ID string `json:"id"`
	}
	if adminProposal.Code != http.StatusCreated || json.Unmarshal(adminProposal.Body.Bytes(), &adminCandidate) != nil {
		t.Fatalf("admin proposal = %d %s", adminProposal.Code, adminProposal.Body.String())
	}
	adminDenied := runtimeRequest(fixture.runtime, http.MethodPost, "/api/knowledge/candidates/"+adminCandidate.ID+"/review", `{"action":"approve","expected_revision":1,"rationale":"documented admin emergency","emergency":true}`, adminHeaders)
	if adminDenied.Code != http.StatusForbidden {
		t.Fatalf("admin emergency self review = %d %s", adminDenied.Code, adminDenied.Body.String())
	}
}
