package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

type collaborationRuntimeFixture struct {
	runtime                   *Runtime
	config                    Config
	login                     runtimeLogin
	workspaceID, issueID      string
	identifier, workspaceSlug string
	headers                   map[string]string
}

func TestSQLiteRuntimeServesIssueCollaborationRoutes(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-collaboration.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "issue-collaboration@example.com")
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Issue Collaboration","slug":"issue-collaboration"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	})
	if workspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := map[string]string{
		"Authorization": "Bearer " + login.Token, "X-Workspace-Slug": "issue-collaboration", "Content-Type": "application/json",
	}
	created := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Collaborate"}`, headers)
	if created.Code != http.StatusCreated {
		t.Fatalf("create Issue = %d %s", created.Code, created.Body.String())
	}
	var issue struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &issue); err != nil || issue.ID == "" {
		t.Fatalf("decode Issue: %v body=%s", err, created.Body.String())
	}

	for _, probe := range []struct {
		name, method, path, body string
	}{
		{name: "timeline", method: http.MethodGet, path: "/api/issues/" + issue.ID + "/timeline"},
		{name: "comment", method: http.MethodPost, path: "/api/issues/" + issue.ID + "/comments", body: `{"content":"first"}`},
		{name: "issue reaction", method: http.MethodPost, path: "/api/issues/" + issue.ID + "/reactions", body: `{"emoji":"👍"}`},
		{name: "subscribers", method: http.MethodGet, path: "/api/issues/" + issue.ID + "/subscribers"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			response := runtimeRequest(runtime, probe.method, probe.path, probe.body, headers)
			if response.Code == http.StatusNotFound {
				t.Fatalf("collaboration route is missing: %s %s = %d %s", probe.method, probe.path, response.Code, response.Body.String())
			}
		})
	}
	config := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if config.Code != http.StatusOK || !containsJSON(config.Body.Bytes(), `"issue_timeline":true`, `"issue_reactions":true`, `"issue_subscribers":true`) {
		t.Fatalf("collaboration capabilities = %d %s", config.Code, config.Body.String())
	}
}

func TestSQLiteRuntimeHonorsIssueCollaborationContract(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-collaboration-contract.db"), "collaboration-contract", "collaboration-owner@example.com")
	member := addCollaborationRuntimeMember(t, fixture, "collaboration-member@example.com", "member")
	otherMember := addCollaborationRuntimeMember(t, fixture, "collaboration-other@example.com", "member")
	admin := addCollaborationRuntimeMember(t, fixture, "collaboration-admin@example.com", "admin")
	outsider := verifyRuntimeLogin(t, fixture.runtime, "collaboration-outsider@example.com")

	missingAuth := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/comments", "", map[string]string{"X-Workspace-Slug": fixture.workspaceSlug})
	assertRuntimeResponse(t, missingAuth.Code, missingAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	missingWorkspace := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/comments", "", map[string]string{"Authorization": "Bearer " + fixture.login.Token})
	assertRuntimeResponse(t, missingWorkspace.Code, missingWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)
	foreign := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/comments", "", collaborationHeaders(outsider.Token, fixture.workspaceSlug))
	assertRuntimeResponse(t, foreign.Code, foreign.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	if _, err := fixture.runtime.Database().Exec(`UPDATE auth_sessions SET expires_at_unix_nano=0 WHERE user_id=?`, outsider.UserID); err != nil {
		t.Fatal(err)
	}
	expired := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/comments", "", collaborationHeaders(outsider.Token, fixture.workspaceSlug))
	assertRuntimeResponse(t, expired.Code, expired.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	secondWorkspace := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Second Workspace","slug":"collaboration-second"}`, map[string]string{"Authorization": "Bearer " + fixture.login.Token, "Content-Type": "application/json"})
	var secondWorkspaceBody struct {
		ID string `json:"id"`
	}
	if secondWorkspace.Code != http.StatusCreated || json.Unmarshal(secondWorkspace.Body.Bytes(), &secondWorkspaceBody) != nil {
		t.Fatalf("second workspace = %d %s", secondWorkspace.Code, secondWorkspace.Body.String())
	}
	mismatchHeaders := collaborationHeaders(fixture.login.Token, fixture.workspaceSlug)
	mismatchHeaders["X-Workspace-ID"] = secondWorkspaceBody.ID
	mismatch := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/comments", "", mismatchHeaders)
	assertRuntimeResponse(t, mismatch.Code, mismatch.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)

	cookieWithoutCSRF := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":"blocked"}`, map[string]string{
		"Cookie": "multica_auth=" + fixture.login.Token, "X-Workspace-Slug": fixture.workspaceSlug, "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, cookieWithoutCSRF.Code, cookieWithoutCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	missingMalformed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/missing/comments", `{"unknown":true}`, fixture.headers)
	assertRuntimeResponse(t, missingMalformed.Code, missingMalformed.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	for name, body := range map[string]string{
		"unknown":   `{"content":"bad","unknown":true}`,
		"trailing":  `{"content":"bad"} {}`,
		"oversized": `{"content":"` + strings.Repeat("x", (1<<20)+1) + `"}`,
	} {
		response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", body, fixture.headers)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s comment body = %d %s", name, response.Code, response.Body.String())
		}
	}

	root := createRuntimeComment(t, fixture.runtime, fixture.headers, fixture.issueID, `{"content":"root decision"}`)
	memberHeaders := collaborationHeaders(member.Token, fixture.workspaceSlug)
	reply := createRuntimeComment(t, fixture.runtime, memberHeaders, fixture.identifier, `{"content":"reply decision","parent_id":"`+root.ID+`"}`)
	if reply.ParentID == nil || *reply.ParentID != root.ID || reply.AuthorID != member.UserID {
		t.Fatalf("reply = %#v", reply)
	}
	secondIssue := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Second collaboration Issue")
	foreignParent := createRuntimeComment(t, fixture.runtime, fixture.headers, secondIssue.ID, `{"content":"other issue"}`)
	badParent := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":"bad parent","parent_id":"`+foreignParent.ID+`"}`, fixture.headers)
	assertRuntimeResponse(t, badParent.Code, badParent.Body.String(), http.StatusBadRequest, `{"error":"invalid request"}`)

	listed := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.identifier+"/comments", "", fixture.headers)
	var comments []runtimeCommentResponse
	if listed.Code != http.StatusOK || json.Unmarshal(listed.Body.Bytes(), &comments) != nil || len(comments) != 2 || comments[0].ID != root.ID || comments[1].ID != reply.ID {
		t.Fatalf("comments = %d %s", listed.Code, listed.Body.String())
	}
	for name, request := range map[string]*httptest.ResponseRecorder{
		"update":     runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+reply.ID, `{"content":"blocked"}`, map[string]string{"Cookie": "multica_auth=" + fixture.login.Token, "X-Workspace-Slug": fixture.workspaceSlug, "Content-Type": "application/json"}),
		"reaction":   runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"blocked"}`, map[string]string{"Cookie": "multica_auth=" + fixture.login.Token, "X-Workspace-Slug": fixture.workspaceSlug, "Content-Type": "application/json"}),
		"subscriber": runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/subscribe", `{}`, map[string]string{"Cookie": "multica_auth=" + fixture.login.Token, "X-Workspace-Slug": fixture.workspaceSlug, "Content-Type": "application/json"}),
	} {
		if request.Code != http.StatusForbidden || strings.TrimSpace(request.Body.String()) != `{"error":"invalid CSRF token"}` {
			t.Fatalf("cookie %s without CSRF = %d %s", name, request.Code, request.Body.String())
		}
	}
	cookieHeaders := map[string]string{
		"Cookie":       "multica_auth=" + fixture.login.Token + "; multica_csrf=" + fixture.login.CSRF,
		"X-CSRF-Token": fixture.login.CSRF, "X-Workspace-Slug": fixture.workspaceSlug, "Content-Type": "application/json",
	}
	cookieComment := createRuntimeComment(t, fixture.runtime, cookieHeaders, fixture.issueID, `{"content":"cookie comment"}`)
	if cookieComment.AuthorID != fixture.login.UserID {
		t.Fatalf("cookie comment actor = %s", cookieComment.AuthorID)
	}
	for name, request := range map[string]*httptest.ResponseRecorder{
		"unknown reaction":   runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"x","unknown":true}`, fixture.headers),
		"unknown subscriber": runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/subscribe", `{"unknown":true}`, fixture.headers),
	} {
		if request.Code != http.StatusBadRequest {
			t.Fatalf("%s = %d %s", name, request.Code, request.Body.String())
		}
	}
	missingCommentMalformed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/missing/reactions", `{"unknown":true}`, fixture.headers)
	assertRuntimeResponse(t, missingCommentMalformed.Code, missingCommentMalformed.Body.String(), http.StatusNotFound, `{"error":"comment not found"}`)
	missingIssueMalformed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/missing/reactions", `{"unknown":true}`, fixture.headers)
	assertRuntimeResponse(t, missingIssueMalformed.Code, missingIssueMalformed.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)

	updated := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+reply.ID, `{"content":"member edit","attachment_ids":[]}`, memberHeaders)
	if updated.Code != http.StatusOK || !containsJSON(updated.Body.Bytes(), `"content":"member edit"`) {
		t.Fatalf("member update = %d %s", updated.Code, updated.Body.String())
	}
	denied := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+reply.ID, `{"content":"other edit"}`, collaborationHeaders(otherMember.Token, fixture.workspaceSlug))
	assertRuntimeResponse(t, denied.Code, denied.Body.String(), http.StatusForbidden, `{"error":"insufficient permissions"}`)
	moderated := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+reply.ID, `{"content":"admin edit"}`, collaborationHeaders(admin.Token, fixture.workspaceSlug))
	if moderated.Code != http.StatusOK || !containsJSON(moderated.Body.Bytes(), `"content":"admin edit"`) {
		t.Fatalf("admin update = %d %s", moderated.Code, moderated.Body.String())
	}
	missingUpdate := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/missing", `{"unknown":true}`, fixture.headers)
	assertRuntimeResponse(t, missingUpdate.Code, missingUpdate.Body.String(), http.StatusNotFound, `{"error":"comment not found"}`)

	resolvedRoot := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+root.ID+"/resolve", "", fixture.headers)
	if resolvedRoot.Code != http.StatusOK || !containsJSON(resolvedRoot.Body.Bytes(), `"resolved_by_id":"`+fixture.login.UserID+`"`) {
		t.Fatalf("resolve root = %d %s", resolvedRoot.Code, resolvedRoot.Body.String())
	}
	resolvedReply := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/resolve", "", memberHeaders)
	if resolvedReply.Code != http.StatusOK || !containsJSON(resolvedReply.Body.Bytes(), `"resolved_by_id":"`+member.UserID+`"`) {
		t.Fatalf("resolve reply = %d %s", resolvedReply.Code, resolvedReply.Body.String())
	}
	listed = runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/comments", "", fixture.headers)
	if json.Unmarshal(listed.Body.Bytes(), &comments) != nil || comments[0].ResolvedAt != nil || comments[1].ResolvedAt == nil {
		t.Fatalf("single thread resolution = %s", listed.Body.String())
	}
	unresolved := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+reply.ID+"/resolve", "", memberHeaders)
	if unresolved.Code != http.StatusOK || !containsJSON(unresolved.Body.Bytes(), `"resolved_at":null`) {
		t.Fatalf("unresolve = %d %s", unresolved.Code, unresolved.Body.String())
	}

	proposal := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/knowledge-proposals", "", memberHeaders)
	if proposal.Code != http.StatusAccepted || !containsJSON(proposal.Body.Bytes(), `"queued":true`, `"evidence_id":`) || !strings.Contains(proposal.Body.String(), `@sha256:`) {
		t.Fatalf("knowledge proposal = %d %s", proposal.Code, proposal.Body.String())
	}
	repeatedProposal := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/knowledge-proposals", "", memberHeaders)
	if repeatedProposal.Code != http.StatusOK || !containsJSON(repeatedProposal.Body.Bytes(), `"queued":false`, `"evidence_id":null`) {
		t.Fatalf("repeated knowledge proposal = %d %s", repeatedProposal.Code, repeatedProposal.Body.String())
	}
	editForRevision := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+reply.ID, `{"content":"new revision"}`, memberHeaders)
	if editForRevision.Code != http.StatusOK {
		t.Fatalf("edit for proposal = %d %s", editForRevision.Code, editForRevision.Body.String())
	}
	newProposal := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/knowledge-proposals", "", memberHeaders)
	if newProposal.Code != http.StatusAccepted {
		t.Fatalf("new knowledge revision = %d %s", newProposal.Code, newProposal.Body.String())
	}

	firstReaction := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"👍"}`, memberHeaders)
	secondReaction := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"👍"}`, memberHeaders)
	if firstReaction.Code != http.StatusCreated || secondReaction.Code != http.StatusCreated || firstReaction.Body.String() != secondReaction.Body.String() {
		t.Fatalf("idempotent comment reaction = %d/%d %s / %s", firstReaction.Code, secondReaction.Code, firstReaction.Body.String(), secondReaction.Body.String())
	}
	removeReaction := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"👍"}`, memberHeaders)
	assertRuntimeResponse(t, removeReaction.Code, removeReaction.Body.String(), http.StatusNoContent, ``)

	issueReaction := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/reactions", `{"emoji":"🚀"}`, fixture.headers)
	if issueReaction.Code != http.StatusCreated || !containsJSON(issueReaction.Body.Bytes(), `"issue_id":"`+fixture.issueID+`"`) {
		t.Fatalf("Issue reaction = %d %s", issueReaction.Code, issueReaction.Body.String())
	}
	issueReactions := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.identifier+"/reactions", "", fixture.headers)
	if issueReactions.Code != http.StatusOK || !containsJSON(issueReactions.Body.Bytes(), `"issue_id":"`+fixture.issueID+`"`, `"emoji":"🚀"`) {
		t.Fatalf("Issue reactions = %d %s", issueReactions.Code, issueReactions.Body.String())
	}

	subscribed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/subscribe", `{}`, fixture.headers)
	assertRuntimeResponse(t, subscribed.Code, subscribed.Body.String(), http.StatusOK, `{"subscribed":true}`)
	subscribeMember := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/subscribe", `{"user_type":"member","user_id":"`+member.UserID+`"}`, fixture.headers)
	assertRuntimeResponse(t, subscribeMember.Code, subscribeMember.Body.String(), http.StatusOK, `{"subscribed":true}`)
	subscribeOutsider := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/subscribe", `{"user_type":"member","user_id":"`+outsider.UserID+`"}`, fixture.headers)
	assertRuntimeResponse(t, subscribeOutsider.Code, subscribeOutsider.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	subscribers := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/subscribers", "", fixture.headers)
	if subscribers.Code != http.StatusOK || !containsJSON(subscribers.Body.Bytes(), `"user_id":"`+fixture.login.UserID+`"`, `"user_id":"`+member.UserID+`"`) {
		t.Fatalf("subscribers = %d %s", subscribers.Code, subscribers.Body.String())
	}
	unsubscribed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/unsubscribe", `{"user_type":"member","user_id":"`+member.UserID+`"}`, fixture.headers)
	assertRuntimeResponse(t, unsubscribed.Code, unsubscribed.Body.String(), http.StatusOK, `{"subscribed":false}`)

	statusUpdate := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID, `{"status":"in_progress"}`, fixture.headers)
	if statusUpdate.Code != http.StatusOK {
		t.Fatalf("status update = %d %s", statusUpdate.Code, statusUpdate.Body.String())
	}
	timeline := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.identifier+"/timeline", "", fixture.headers)
	if timeline.Code != http.StatusOK || !containsJSON(timeline.Body.Bytes(), `"type":"comment"`, `"type":"activity"`, `"action":"created"`, `"action":"status_changed"`, `"from":"todo"`, `"to":"in_progress"`) {
		t.Fatalf("timeline = %d %s", timeline.Code, timeline.Body.String())
	}
}

func TestSQLiteRuntimePublishesOnlyCommittedIssueCollaborationEvents(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "issue-collaboration-events.db"), "collaboration-events", "collaboration-events@example.com")
	server := httptest.NewServer(fixture.runtime.HTTPServer())
	t.Cleanup(server.Close)

	createdSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	comment := createRuntimeComment(t, fixture.runtime, fixture.headers, fixture.identifier, `{"content":"event comment"}`)
	assertRealtimeEvent(t, createdSocket, "comment:created", `"issue_id":"`+fixture.issueID+`"`)
	_ = createdSocket.Close()

	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER fail_comment_create BEFORE INSERT ON workspace_issue_comments BEGIN SELECT RAISE(ABORT,'forced comment rollback'); END`); err != nil {
		t.Fatal(err)
	}
	failureSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	failedCreate := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/comments", `{"content":"rolled back"}`, fixture.headers)
	if failedCreate.Code != http.StatusInternalServerError {
		t.Fatalf("failed comment create = %d %s", failedCreate.Code, failedCreate.Body.String())
	}
	assertNoRealtimeEvent(t, failureSocket)
	_ = failureSocket.Close()
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER fail_comment_create`); err != nil {
		t.Fatal(err)
	}

	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER fail_comment_readback AFTER UPDATE OF content ON workspace_issue_comments WHEN NEW.id='` + comment.ID + `' BEGIN DELETE FROM workspace_issue_comments WHERE id=NEW.id; END`); err != nil {
		t.Fatal(err)
	}
	updateFailureSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	failedUpdate := runtimeRequest(fixture.runtime, http.MethodPut, "/api/comments/"+comment.ID, `{"content":"must roll back"}`, fixture.headers)
	if failedUpdate.Code != http.StatusInternalServerError {
		t.Fatalf("failed comment update = %d %s", failedUpdate.Code, failedUpdate.Body.String())
	}
	assertNoRealtimeEvent(t, updateFailureSocket)
	_ = updateFailureSocket.Close()
	var retainedContent string
	if err := fixture.runtime.Database().QueryRow(`SELECT content FROM workspace_issue_comments WHERE id=?`, comment.ID).Scan(&retainedContent); err != nil || retainedContent != "event comment" {
		t.Fatalf("rolled back comment readback = %q err=%v", retainedContent, err)
	}
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER fail_comment_readback`); err != nil {
		t.Fatal(err)
	}

	reactionSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	addedReaction := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/reactions", `{"emoji":"✅"}`, fixture.headers)
	if addedReaction.Code != http.StatusCreated {
		t.Fatalf("add Issue reaction = %d %s", addedReaction.Code, addedReaction.Body.String())
	}
	assertRealtimeEvent(t, reactionSocket, "issue_reaction:added", `"issue_id":"`+fixture.issueID+`"`)
	removedReaction := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/issues/"+fixture.identifier+"/reactions", `{"emoji":"✅"}`, fixture.headers)
	if removedReaction.Code != http.StatusNoContent {
		t.Fatalf("remove Issue reaction = %d %s", removedReaction.Code, removedReaction.Body.String())
	}
	assertRealtimeEvent(t, reactionSocket, "issue_reaction:removed", `"issue_id":"`+fixture.issueID+`"`)
	_ = reactionSocket.Close()

	subscriberSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	subscribed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/subscribe", `{}`, fixture.headers)
	if subscribed.Code != http.StatusOK {
		t.Fatalf("subscribe = %d %s", subscribed.Code, subscribed.Body.String())
	}
	assertRealtimeEvent(t, subscriberSocket, "subscriber:added", `"issue_id":"`+fixture.issueID+`"`)
	unsubscribed := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/unsubscribe", `{}`, fixture.headers)
	if unsubscribed.Code != http.StatusOK {
		t.Fatalf("unsubscribe = %d %s", unsubscribed.Code, unsubscribed.Body.String())
	}
	assertRealtimeEvent(t, subscriberSocket, "subscriber:removed", `"issue_id":"`+fixture.issueID+`"`)
	_ = subscriberSocket.Close()

	deleteRoot := createRuntimeComment(t, fixture.runtime, fixture.headers, fixture.issueID, `{"content":"delete thread"}`)
	deleteReply := createRuntimeComment(t, fixture.runtime, fixture.headers, fixture.issueID, `{"content":"delete reply","parent_id":"`+deleteRoot.ID+`"}`)
	if response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+deleteReply.ID+"/reactions", `{"emoji":"cleanup"}`, fixture.headers); response.Code != http.StatusCreated {
		t.Fatalf("delete-thread reaction = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+deleteReply.ID+"/knowledge-proposals", "", fixture.headers); response.Code != http.StatusAccepted {
		t.Fatalf("delete-thread knowledge proposal = %d %s", response.Code, response.Body.String())
	}
	deleteSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	deletedComment := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/comments/"+deleteRoot.ID, "", fixture.headers)
	assertRuntimeResponse(t, deletedComment.Code, deletedComment.Body.String(), http.StatusNoContent, ``)
	assertRealtimeEvent(t, deleteSocket, "comment:deleted", `"comment_id":"`+deleteRoot.ID+`"`)
	_ = deleteSocket.Close()
	for name, query := range map[string]string{
		"thread comments":  `SELECT COUNT(*) FROM workspace_issue_comments WHERE id IN (?,?)`,
		"thread reactions": `SELECT COUNT(*) FROM workspace_comment_reactions WHERE comment_id IN (?,?)`,
		"thread proposals": `SELECT COUNT(*) FROM workspace_comment_knowledge_proposals WHERE comment_id IN (?,?)`,
	} {
		var count int
		if err := fixture.runtime.Database().QueryRow(query, deleteRoot.ID, deleteReply.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s after root delete = %d err=%v", name, count, err)
		}
	}

	activitySocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	updatedIssue := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID, `{"status":"done"}`, fixture.headers)
	if updatedIssue.Code != http.StatusOK {
		t.Fatalf("update Issue = %d %s", updatedIssue.Code, updatedIssue.Body.String())
	}
	assertRealtimeEvent(t, activitySocket, "issue:updated", `"status":"done"`)
	assertRealtimeEvent(t, activitySocket, "activity:created", `"action":"status_changed"`)
	_ = activitySocket.Close()
}

func TestSQLiteRuntimePersistsSerializesAndCleansIssueCollaboration(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-collaboration-retained.db")
	fixture := newCollaborationRuntimeFixture(t, databasePath, "collaboration-retained", "collaboration-retained@example.com")
	root := createRuntimeComment(t, fixture.runtime, fixture.headers, fixture.issueID, `{"content":"retained root"}`)
	reply := createRuntimeComment(t, fixture.runtime, fixture.headers, fixture.issueID, `{"content":"retained reply","parent_id":"`+root.ID+`"}`)
	if response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"retained"}`, fixture.headers); response.Code != http.StatusCreated {
		t.Fatalf("retained comment reaction = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/reactions", `{"emoji":"retained"}`, fixture.headers); response.Code != http.StatusCreated {
		t.Fatalf("retained Issue reaction = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/subscribe", `{}`, fixture.headers); response.Code != http.StatusOK {
		t.Fatalf("retained subscriber = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/comments/"+reply.ID+"/knowledge-proposals", "", fixture.headers); response.Code != http.StatusAccepted {
		t.Fatalf("retained knowledge = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID, `{"priority":"high"}`, fixture.headers); response.Code != http.StatusOK {
		t.Fatalf("retained activity = %d %s", response.Code, response.Body.String())
	}
	if err := fixture.runtime.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := newRuntimeForConfig(t, fixture.config)
	comments := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.identifier+"/comments", "", fixture.headers)
	if comments.Code != http.StatusOK || !containsJSON(comments.Body.Bytes(), `"content":"retained root"`, `"content":"retained reply"`, `"emoji":"retained"`) {
		t.Fatalf("restart comments = %d %s", comments.Code, comments.Body.String())
	}
	timeline := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.identifier+"/timeline", "", fixture.headers)
	if timeline.Code != http.StatusOK || !containsJSON(timeline.Body.Bytes(), `"action":"created"`, `"action":"priority_changed"`, `"type":"comment"`) {
		t.Fatalf("restart timeline = %d %s", timeline.Code, timeline.Body.String())
	}
	reactions := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.identifier+"/reactions", "", fixture.headers)
	if reactions.Code != http.StatusOK || !containsJSON(reactions.Body.Bytes(), `"emoji":"retained"`) {
		t.Fatalf("restart reactions = %d %s", reactions.Code, reactions.Body.String())
	}
	subscribers := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.identifier+"/subscribers", "", fixture.headers)
	if subscribers.Code != http.StatusOK || !containsJSON(subscribers.Body.Bytes(), `"user_id":"`+fixture.login.UserID+`"`) {
		t.Fatalf("restart subscribers = %d %s", subscribers.Code, subscribers.Body.String())
	}

	const workers = 12
	subscriberStart := make(chan struct{})
	subscriberResults := make(chan *httptest.ResponseRecorder, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-subscriberStart
			subscriberResults <- runtimeRequest(restarted, http.MethodPost, "/api/issues/"+fixture.identifier+"/subscribe", `{}`, fixture.headers)
		}()
	}
	close(subscriberStart)
	group.Wait()
	close(subscriberResults)
	for response := range subscriberResults {
		assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusOK, `{"subscribed":true}`)
	}
	var concurrentSubscriberCount int
	if err := restarted.Database().QueryRow(`SELECT COUNT(*) FROM workspace_issue_subscribers WHERE workspace_id=? AND issue_id=? AND user_type='member' AND user_id=?`, fixture.workspaceID, fixture.issueID, fixture.login.UserID).Scan(&concurrentSubscriberCount); err != nil || concurrentSubscriberCount != 1 {
		t.Fatalf("concurrent subscriber count = %d err=%v", concurrentSubscriberCount, err)
	}

	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, workers)
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			results <- runtimeRequest(restarted, http.MethodPost, "/api/comments/"+reply.ID+"/reactions", `{"emoji":"concurrent"}`, fixture.headers)
		}()
	}
	close(start)
	group.Wait()
	close(results)
	var canonicalReaction string
	for response := range results {
		if response.Code != http.StatusCreated {
			t.Fatalf("concurrent reaction = %d %s", response.Code, response.Body.String())
		}
		if canonicalReaction == "" {
			canonicalReaction = response.Body.String()
		} else if response.Body.String() != canonicalReaction {
			t.Fatalf("concurrent reaction identities diverged: %s / %s", canonicalReaction, response.Body.String())
		}
	}
	var concurrentReactionCount int
	if err := restarted.Database().QueryRow(`SELECT COUNT(*) FROM workspace_comment_reactions WHERE workspace_id=? AND comment_id=? AND emoji='concurrent'`, fixture.workspaceID, reply.ID).Scan(&concurrentReactionCount); err != nil || concurrentReactionCount != 1 {
		t.Fatalf("concurrent reaction count = %d err=%v", concurrentReactionCount, err)
	}

	resolveResults := make(chan int, 2)
	start = make(chan struct{})
	for _, commentID := range []string{root.ID, reply.ID} {
		group.Add(1)
		go func(commentID string) {
			defer group.Done()
			<-start
			resolveResults <- runtimeRequest(restarted, http.MethodPost, "/api/comments/"+commentID+"/resolve", "", fixture.headers).Code
		}(commentID)
	}
	close(start)
	group.Wait()
	close(resolveResults)
	for code := range resolveResults {
		if code != http.StatusOK {
			t.Fatalf("concurrent resolve = %d", code)
		}
	}
	var resolvedCount int
	if err := restarted.Database().QueryRow(`SELECT COUNT(*) FROM workspace_issue_comments WHERE workspace_id=? AND issue_id=? AND resolved_at IS NOT NULL`, fixture.workspaceID, fixture.issueID).Scan(&resolvedCount); err != nil || resolvedCount != 1 {
		t.Fatalf("resolved thread count = %d err=%v", resolvedCount, err)
	}

	deleted := runtimeRequest(restarted, http.MethodDelete, "/api/issues/"+fixture.identifier, "", fixture.headers)
	assertRuntimeResponse(t, deleted.Code, deleted.Body.String(), http.StatusNoContent, ``)
	assertNoCollaborationRows(t, restarted, fixture.workspaceID, fixture.issueID)

	batchIssue := createRuntimeIssue(t, restarted, fixture.headers, "Batch cleanup")
	batchComment := createRuntimeComment(t, restarted, fixture.headers, batchIssue.ID, `{"content":"batch cleanup"}`)
	if response := runtimeRequest(restarted, http.MethodPost, "/api/comments/"+batchComment.ID+"/reactions", `{"emoji":"batch"}`, fixture.headers); response.Code != http.StatusCreated {
		t.Fatalf("batch reaction = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(restarted, http.MethodPost, "/api/issues/"+batchIssue.ID+"/subscribe", `{}`, fixture.headers); response.Code != http.StatusOK {
		t.Fatalf("batch subscriber = %d %s", response.Code, response.Body.String())
	}
	batchDeleted := runtimeRequest(restarted, http.MethodPost, "/api/issues/batch-delete", `{"issue_ids":["`+batchIssue.ID+`"]}`, fixture.headers)
	if batchDeleted.Code != http.StatusOK || !containsJSON(batchDeleted.Body.Bytes(), `"deleted":1`) {
		t.Fatalf("batch delete = %d %s", batchDeleted.Code, batchDeleted.Body.String())
	}
	assertNoCollaborationRows(t, restarted, fixture.workspaceID, batchIssue.ID)
}

type runtimeIssueResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
}

type runtimeCommentResponse struct {
	ID           string  `json:"id"`
	IssueID      string  `json:"issue_id"`
	AuthorID     string  `json:"author_id"`
	Content      string  `json:"content"`
	ParentID     *string `json:"parent_id"`
	ResolvedAt   *string `json:"resolved_at"`
	ResolvedByID *string `json:"resolved_by_id"`
}

func newCollaborationRuntimeFixture(t *testing.T, databasePath, slug, email string) collaborationRuntimeFixture {
	t.Helper()
	config := Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: databasePath, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	login := verifyRuntimeLogin(t, runtime, email)
	workspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", fmt.Sprintf(`{"name":"Collaboration %s","slug":%q}`, slug, slug), map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	})
	var workspaceBody struct {
		ID string `json:"id"`
	}
	if workspace.Code != http.StatusCreated || json.Unmarshal(workspace.Body.Bytes(), &workspaceBody) != nil || workspaceBody.ID == "" {
		t.Fatalf("create collaboration Workspace = %d %s", workspace.Code, workspace.Body.String())
	}
	headers := collaborationHeaders(login.Token, slug)
	issue := createRuntimeIssue(t, runtime, headers, "Collaboration Issue")
	return collaborationRuntimeFixture{runtime: runtime, config: config, login: login, workspaceID: workspaceBody.ID, issueID: issue.ID, identifier: issue.Identifier, workspaceSlug: slug, headers: headers}
}

func addCollaborationRuntimeMember(t *testing.T, fixture collaborationRuntimeFixture, email, role string) runtimeLogin {
	t.Helper()
	login := verifyRuntimeLogin(t, fixture.runtime, email)
	memberID := "member-row-" + strings.ReplaceAll(login.UserID, "-", "")
	if _, err := fixture.runtime.Database().Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)`, memberID, fixture.workspaceID, login.UserID, role, "2026-08-14T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	return login
}

func collaborationHeaders(token, slug string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token, "X-Workspace-Slug": slug, "Content-Type": "application/json"}
}

func createRuntimeIssue(t *testing.T, runtime *Runtime, headers map[string]string, title string) runtimeIssueResponse {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodPost, "/api/issues", fmt.Sprintf(`{"title":%q}`, title), headers)
	var issue runtimeIssueResponse
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &issue) != nil || issue.ID == "" || issue.Identifier == "" {
		t.Fatalf("create runtime Issue = %d %s", response.Code, response.Body.String())
	}
	return issue
}

func createRuntimeComment(t *testing.T, runtime *Runtime, headers map[string]string, issueID, body string) runtimeCommentResponse {
	t.Helper()
	response := runtimeRequest(runtime, http.MethodPost, "/api/issues/"+issueID+"/comments", body, headers)
	var comment runtimeCommentResponse
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &comment) != nil || comment.ID == "" {
		t.Fatalf("create runtime comment = %d %s", response.Code, response.Body.String())
	}
	return comment
}

func assertNoCollaborationRows(t *testing.T, runtime *Runtime, workspaceID, issueID string) {
	t.Helper()
	for table, query := range map[string]string{
		"comments":        `SELECT COUNT(*) FROM workspace_issue_comments WHERE workspace_id=? AND issue_id=?`,
		"Issue reactions": `SELECT COUNT(*) FROM workspace_issue_reactions WHERE workspace_id=? AND issue_id=?`,
		"subscribers":     `SELECT COUNT(*) FROM workspace_issue_subscribers WHERE workspace_id=? AND issue_id=?`,
		"activities":      `SELECT COUNT(*) FROM workspace_issue_activities WHERE workspace_id=? AND issue_id=?`,
	} {
		var count int
		if err := runtime.Database().QueryRow(query, workspaceID, issueID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s after Issue deletion = %d err=%v", table, count, err)
		}
	}
	for table, query := range map[string]string{
		"comment reactions":   `SELECT COUNT(*) FROM workspace_comment_reactions WHERE workspace_id=?`,
		"knowledge proposals": `SELECT COUNT(*) FROM workspace_comment_knowledge_proposals WHERE workspace_id=?`,
	} {
		var count int
		if err := runtime.Database().QueryRow(query, workspaceID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s after Issue deletion = %d err=%v", table, count, err)
		}
	}
}
