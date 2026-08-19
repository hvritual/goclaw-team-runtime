package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/workspace"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestSQLiteRuntimePublishesAuthorizedCommittedIssueEvents(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "realtime.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888", SessionTTL: time.Hour},
	})
	owner := verifyRuntimeLogin(t, runtime, "owner@example.com")
	outsider := verifyRuntimeLogin(t, runtime, "outsider@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES
			('workspace-one','One','one','{}','[]','ONE','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z'),
			('workspace-two','Two','two','{}','[]','TWO','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-owner','workspace-one','` + owner.UserID + `','owner','2026-08-13T00:00:00Z'),
			('member-outsider','workspace-two','` + outsider.UserID + `','owner','2026-08-13T00:00:00Z')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	server := httptest.NewServer(runtime.HTTPServer())
	t.Cleanup(server.Close)
	badCookieHeader := http.Header{"Cookie": []string{"multica_auth=invalid"}, "Origin": []string{"http://localhost:3000"}}
	if connection, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws?workspace_slug=one", badCookieHeader); err == nil {
		_ = connection.Close()
		t.Fatal("invalid cookie unexpectedly connected")
	} else if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid cookie response = %#v, %v", response, err)
	} else if body, _ := io.ReadAll(response.Body); strings.TrimSpace(string(body)) != `{"error":"user not authenticated"}` {
		t.Fatalf("invalid cookie body=%s", body)
	}
	assertCookieRealtimeUpgradeDenied(t, server.URL, "", owner.Token, http.StatusNotFound)
	assertCookieRealtimeUpgradeDenied(t, server.URL, "one", outsider.Token, http.StatusNotFound)
	assertTokenRealtimeDenied(t, server.URL, "one", `{"type":"bad"}`)
	assertTokenRealtimeDenied(t, server.URL, "one", `{"type":"auth","payload":{"token":"invalid"}}`)
	assertTokenRealtimeDenied(t, server.URL, "", `{"type":"auth","payload":{"token":"`+owner.Token+`"}}`)
	headerBypass, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws?workspace_slug=one", http.Header{"Origin": []string{"http://localhost:3000"}, "Authorization": []string{"Bearer " + owner.Token}})
	if err != nil {
		t.Fatal(err)
	}
	_ = headerBypass.SetReadDeadline(time.Now().Add(75 * time.Millisecond))
	if _, data, err := headerBypass.ReadMessage(); err == nil {
		t.Fatalf("Authorization header bypassed token first frame: %s", data)
	}
	_ = headerBypass.Close()
	ownerSocket := dialTokenRealtime(t, server.URL, "one", owner.Token)
	t.Cleanup(func() { _ = ownerSocket.Close() })
	oversizedSocket := dialTokenRealtime(t, server.URL, "one", owner.Token)
	if err := oversizedSocket.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("x", 70*1024))); err != nil {
		t.Fatal(err)
	}
	_ = oversizedSocket.SetReadDeadline(time.Now().Add(time.Second))
	if _, data, err := oversizedSocket.ReadMessage(); err == nil {
		t.Fatalf("oversized post-auth frame kept connection open: %s", data)
	}
	_ = oversizedSocket.Close()
	cookieSocket := dialCookieRealtime(t, server.URL, "one", owner.Token)
	t.Cleanup(func() { _ = cookieSocket.Close() })
	foreignSocket := dialTokenRealtime(t, server.URL, "two", outsider.Token)
	t.Cleanup(func() { _ = foreignSocket.Close() })
	assertTokenRealtimeDenied(t, server.URL, "one", `{"type":"auth","payload":{"token":"`+outsider.Token+`"}}`)

	var module *workspace.Module
	for _, candidate := range runtime.Application().Modules() {
		if typed, ok := candidate.(*workspace.Module); ok {
			module = typed
		}
	}
	if module == nil {
		t.Fatal("Workspace module not composed")
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-owner")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-one", Title: "Realtime issue"})
	if err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:created", `"identifier":"ONE-1"`)
	assertRealtimeEvent(t, cookieSocket, "issue:created", `"identifier":"ONE-1"`)
	assertRealtimeEvent(t, ownerSocket, "activity:created", `"action":"created"`)
	assertRealtimeEvent(t, cookieSocket, "activity:created", `"action":"created"`)
	_ = cookieSocket.Close()
	assertNoRealtimeEvent(t, foreignSocket)

	title := "Updated after commit"
	if _, err := module.IssueLocal().UpdateIssue(ctx, contract.UpdateIssueRequest{WorkspaceId: "workspace-one", IssueId: created.Issue.Id, Title: &title}); err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:updated", `"title":"Updated after commit"`)
	assertRealtimeEvent(t, ownerSocket, "activity:created", `"action":"title_changed"`)
	if _, err := module.IssueLocal().UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-one", IssueId: created.Issue.Id, Status: "todo"}); err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:updated", `"status_changed":false`)
	if _, err := module.IssueLocal().UpdateIssueStatus(ctx, contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-one", IssueId: created.Issue.Id, Status: "done"}); err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:updated", `"status_changed":true`)
	assertRealtimeEvent(t, ownerSocket, "activity:created", `"action":"status_changed"`)

	if _, err := module.IssueMetadataLocal().PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-one", IssueId: created.Issue.Id, Key: "complete", ValueJson: `true`}); err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue_metadata:changed", `"metadata":{"complete":true}`)

	failureSocket := dialTokenRealtime(t, server.URL, "one", owner.Token)
	if _, err := module.IssueMetadataLocal().PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-one", IssueId: "missing", Key: "failed", ValueJson: `true`}); err == nil {
		t.Fatal("expected failed transaction")
	}
	assertNoRealtimeEvent(t, failureSocket)
	_ = failureSocket.Close()

	failedDeleteIssue, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-one", Title: "Delete rollback"})
	if err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:created", `"id":"`+failedDeleteIssue.Issue.Id+`"`)
	assertRealtimeEvent(t, ownerSocket, "activity:created", `"action":"created"`)
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO workspace_todos(id,workspace_id,title,status,issue_id,created_at,updated_at) VALUES('todo-rollback','workspace-one','Rollback todo','todo',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{failedDeleteIssue.Issue.Id}},
		{`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,parent_issue_id,created_at,updated_at) VALUES('child-rollback','workspace-one',90,'ONE-90','Rollback child','todo','none','member','member-owner',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{failedDeleteIssue.Issue.Id}},
		{`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-rollback','workspace-one','project-rollback','Rollback requirement',1,'draft','covered',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{`["` + failedDeleteIssue.Issue.Id + `","retained-on-rollback"]`}},
		{`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES('requirement-rollback-v1','requirement-rollback',1,'rollback content','2026-08-13T00:00:00Z')`, nil},
	} {
		if _, err := runtime.Database().Exec(statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	deleteFailureSocket := dialTokenRealtime(t, server.URL, "one", owner.Token)
	if _, err := runtime.Database().Exec(`CREATE TRIGGER fail_realtime_issue_delete BEFORE DELETE ON workspace_issues BEGIN SELECT RAISE(ABORT,'forced delete rollback'); END`); err != nil {
		t.Fatal(err)
	}
	failedDelete := runtimeRequest(runtime, http.MethodDelete, "/api/issues/"+failedDeleteIssue.Issue.Id, "", map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "one"})
	if failedDelete.Code != http.StatusInternalServerError || strings.TrimSpace(failedDelete.Body.String()) != `{"error":"failed to delete issue"}` {
		t.Fatalf("failed delete=%d %s", failedDelete.Code, failedDelete.Body.String())
	}
	assertNoRealtimeEvent(t, deleteFailureSocket)
	_ = deleteFailureSocket.Close()
	retained := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+failedDeleteIssue.Issue.Id, "", map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "one"})
	if retained.Code != http.StatusOK {
		t.Fatalf("rolled back delete=%d %s", retained.Code, retained.Body.String())
	}
	var rollbackTodoIssueID, rollbackParentIssueID, rollbackRequirementIssueIDs string
	var rollbackRequirementVersion int
	if err := runtime.Database().QueryRow(`SELECT issue_id FROM workspace_todos WHERE id='todo-rollback'`).Scan(&rollbackTodoIssueID); err != nil || rollbackTodoIssueID != failedDeleteIssue.Issue.Id {
		t.Fatalf("rolled back todo reference=%q err=%v", rollbackTodoIssueID, err)
	}
	if err := runtime.Database().QueryRow(`SELECT parent_issue_id FROM workspace_issues WHERE id='child-rollback'`).Scan(&rollbackParentIssueID); err != nil || rollbackParentIssueID != failedDeleteIssue.Issue.Id {
		t.Fatalf("rolled back child reference=%q err=%v", rollbackParentIssueID, err)
	}
	if err := runtime.Database().QueryRow(`SELECT issue_ids FROM workspace_requirements WHERE id='requirement-rollback'`).Scan(&rollbackRequirementIssueIDs); err != nil || rollbackRequirementIssueIDs != `["`+failedDeleteIssue.Issue.Id+`","retained-on-rollback"]` {
		t.Fatalf("rolled back requirement references=%q err=%v", rollbackRequirementIssueIDs, err)
	}
	if err := runtime.Database().QueryRow(`SELECT current_version FROM workspace_requirements WHERE id='requirement-rollback'`).Scan(&rollbackRequirementVersion); err != nil || rollbackRequirementVersion != 1 {
		t.Fatalf("rolled back requirement version=%d err=%v", rollbackRequirementVersion, err)
	}
	var rollbackVersionRows int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id='requirement-rollback'`).Scan(&rollbackVersionRows); err != nil || rollbackVersionRows != 1 {
		t.Fatalf("rolled back requirement version rows=%d err=%v", rollbackVersionRows, err)
	}
	if _, err := runtime.Database().Exec(`DROP TRIGGER fail_realtime_issue_delete`); err != nil {
		t.Fatal(err)
	}

	foreignDelete := runtimeRequest(runtime, http.MethodDelete, "/api/issues/"+created.Issue.Id, "", map[string]string{"Authorization": "Bearer " + outsider.Token, "X-Workspace-Slug": "one"})
	if foreignDelete.Code != http.StatusNotFound {
		t.Fatalf("foreign delete=%d %s", foreignDelete.Code, foreignDelete.Body.String())
	}
	cookieDelete := runtimeRequest(runtime, http.MethodDelete, "/api/issues/"+created.Issue.Id, "", map[string]string{"Cookie": "multica_auth=" + owner.Token, "X-Workspace-Slug": "one"})
	if cookieDelete.Code != http.StatusForbidden {
		t.Fatalf("cookie delete=%d %s", cookieDelete.Code, cookieDelete.Body.String())
	}
	missingDeleteAuth := runtimeRequest(runtime, http.MethodDelete, "/api/issues/"+created.Issue.Id, "", map[string]string{"X-Workspace-Slug": "one"})
	if missingDeleteAuth.Code != http.StatusUnauthorized || strings.TrimSpace(missingDeleteAuth.Body.String()) != `{"error":"user not authenticated"}` {
		t.Fatalf("missing delete auth=%d %s", missingDeleteAuth.Code, missingDeleteAuth.Body.String())
	}
	missingDeleteWorkspace := runtimeRequest(runtime, http.MethodDelete, "/api/issues/"+created.Issue.Id, "", map[string]string{"Authorization": "Bearer " + owner.Token})
	if missingDeleteWorkspace.Code != http.StatusBadRequest || strings.TrimSpace(missingDeleteWorkspace.Body.String()) != `{"error":"workspace_id is required"}` {
		t.Fatalf("missing delete workspace=%d %s", missingDeleteWorkspace.Code, missingDeleteWorkspace.Body.String())
	}
	missingDeleteIssue := runtimeRequest(runtime, http.MethodDelete, "/api/issues/missing", "", map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "one"})
	if missingDeleteIssue.Code != http.StatusNotFound || strings.TrimSpace(missingDeleteIssue.Body.String()) != `{"error":"issue not found"}` {
		t.Fatalf("missing delete issue=%d %s", missingDeleteIssue.Code, missingDeleteIssue.Body.String())
	}
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,created_at,updated_at) VALUES('retained-issue','workspace-one',91,'ONE-91','Retained issue','todo','none','member','member-owner','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, nil},
		{`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,parent_issue_id,created_at,updated_at) VALUES('child-success','workspace-one',92,'ONE-92','Child','todo','none','member','member-owner','ONE-1','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, nil},
		{`INSERT INTO workspace_todos(id,workspace_id,title,status,issue_id,created_at,updated_at) VALUES('todo-success','workspace-one','Linked todo','todo','ONE-1','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, nil},
		{`INSERT INTO workspace_todos(id,workspace_id,title,status,issue_id,created_at,updated_at) VALUES('todo-foreign','workspace-two','Foreign todo','todo',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{created.Issue.Id}},
		{`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-success','workspace-one','project-success','Requirement',1,'draft','covered',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{`["` + created.Issue.Id + `","ONE-1","retained-issue"]`}},
		{`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES('requirement-success-v1','requirement-success',1,'success content','2026-08-13T00:00:00Z')`, nil},
		{`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-last-link','workspace-one','project-success','Last link',1,'draft','covered',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{`["` + created.Issue.Id + `"]`}},
		{`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES('requirement-last-link-v1','requirement-last-link',1,'last-link content','2026-08-13T00:00:00Z')`, nil},
		{`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-foreign','workspace-two','project-foreign','Foreign requirement',1,'draft','covered',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, []any{`["` + created.Issue.Id + `"]`}},
	} {
		if _, err := runtime.Database().Exec(statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	deleted := runtimeRequest(runtime, http.MethodDelete, "/api/issues/ONE-1", "", map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "one"})
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete=%d %s", deleted.Code, deleted.Body.String())
	}
	assertRealtimeEvent(t, ownerSocket, "issue:deleted", `"issue_id":"`+created.Issue.Id+`"`)
	afterDelete := runtimeRequest(runtime, http.MethodGet, "/api/issues/"+created.Issue.Id, "", map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "one"})
	if afterDelete.Code != http.StatusNotFound {
		t.Fatalf("after delete=%d %s", afterDelete.Code, afterDelete.Body.String())
	}
	var childParent, todoIssue, requirementIssueIDs *string
	if err := runtime.Database().QueryRow(`SELECT parent_issue_id FROM workspace_issues WHERE id='child-success'`).Scan(&childParent); err != nil || childParent != nil {
		t.Fatalf("child reference after delete=%v err=%v", childParent, err)
	}
	if err := runtime.Database().QueryRow(`SELECT issue_id FROM workspace_todos WHERE id='todo-success'`).Scan(&todoIssue); err != nil || todoIssue != nil {
		t.Fatalf("todo reference after delete=%v err=%v", todoIssue, err)
	}
	wantRequirementIssueIDs := `["` + created.Issue.Id + `","ONE-1","retained-issue"]`
	if err := runtime.Database().QueryRow(`SELECT issue_ids FROM workspace_requirements WHERE id='requirement-success'`).Scan(&requirementIssueIDs); err != nil || requirementIssueIDs == nil || *requirementIssueIDs != wantRequirementIssueIDs {
		t.Fatalf("requirement references after delete=%v err=%v", requirementIssueIDs, err)
	}
	var coverage, updatedAt, currentContent string
	var currentVersion int
	if err := runtime.Database().QueryRow(`SELECT current_version,coverage_status,updated_at FROM workspace_requirements WHERE id='requirement-last-link'`).Scan(&currentVersion, &coverage, &updatedAt); err != nil || currentVersion != 1 || coverage != "covered" || updatedAt != "2026-08-13T00:00:00Z" {
		t.Fatalf("last-link lifecycle version=%d coverage=%q updated=%q err=%v", currentVersion, coverage, updatedAt, err)
	}
	if err := runtime.Database().QueryRow(`SELECT content FROM workspace_requirement_versions WHERE requirement_id='requirement-last-link' AND version=1`).Scan(&currentContent); err != nil || currentContent != "last-link content" {
		t.Fatalf("last-link audit content=%q err=%v", currentContent, err)
	}
	var requirementVersionCount int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id IN ('requirement-success','requirement-last-link')`).Scan(&requirementVersionCount); err != nil || requirementVersionCount != 2 {
		t.Fatalf("legacy requirement version count=%d err=%v", requirementVersionCount, err)
	}
	var foreignTodoIssueID, foreignRequirementIssueIDs string
	if err := runtime.Database().QueryRow(`SELECT issue_id FROM workspace_todos WHERE id='todo-foreign'`).Scan(&foreignTodoIssueID); err != nil || foreignTodoIssueID != created.Issue.Id {
		t.Fatalf("foreign todo reference=%q err=%v", foreignTodoIssueID, err)
	}
	if err := runtime.Database().QueryRow(`SELECT issue_ids FROM workspace_requirements WHERE id='requirement-foreign'`).Scan(&foreignRequirementIssueIDs); err != nil || foreignRequirementIssueIDs != `["`+created.Issue.Id+`"]` {
		t.Fatalf("foreign requirement references=%q err=%v", foreignRequirementIssueIDs, err)
	}
	concurrentIssue, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-one", Title: "Concurrent delete"})
	if err != nil {
		t.Fatal(err)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:created", `"id":"`+concurrentIssue.Issue.Id+`"`)
	assertRealtimeEvent(t, ownerSocket, "activity:created", `"action":"created"`)
	if _, err := runtime.Database().Exec(`INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES('requirement-concurrent','workspace-one','project-success','Concurrent requirement',1,'draft','covered',?,'2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`, `["`+concurrentIssue.Issue.Id+`"]`); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES('requirement-concurrent-v1','requirement-concurrent',1,'concurrent content','2026-08-13T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	type concurrentDeleteResponse struct {
		status int
		body   string
	}
	responses := make(chan concurrentDeleteResponse, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			response := runtimeRequest(runtime, http.MethodDelete, "/api/issues/"+concurrentIssue.Issue.Id, "", map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "one"})
			responses <- concurrentDeleteResponse{status: response.Code, body: response.Body.String()}
		}()
	}
	wait.Wait()
	close(responses)
	statusCounts := map[int]int{}
	var responseBodies []string
	for response := range responses {
		statusCounts[response.status]++
		responseBodies = append(responseBodies, response.body)
	}
	if statusCounts[http.StatusNoContent] != 1 || statusCounts[http.StatusNotFound] != 1 {
		t.Fatalf("concurrent delete statuses=%v bodies=%q", statusCounts, responseBodies)
	}
	assertRealtimeEvent(t, ownerSocket, "issue:deleted", `"issue_id":"`+concurrentIssue.Issue.Id+`"`)
	assertNoRealtimeEvent(t, ownerSocket)
	var concurrentVersion int
	if err := runtime.Database().QueryRow(`SELECT current_version FROM workspace_requirements WHERE id='requirement-concurrent'`).Scan(&concurrentVersion); err != nil || concurrentVersion != 1 {
		t.Fatalf("concurrent requirement version=%d err=%v", concurrentVersion, err)
	}
	var concurrentVersionRows int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_requirement_versions WHERE requirement_id='requirement-concurrent'`).Scan(&concurrentVersionRows); err != nil || concurrentVersionRows != 1 {
		t.Fatalf("concurrent requirement audit rows=%d err=%v", concurrentVersionRows, err)
	}
	var concurrentIssueIDs string
	if err := runtime.Database().QueryRow(`SELECT issue_ids FROM workspace_requirements WHERE id='requirement-concurrent'`).Scan(&concurrentIssueIDs); err != nil || concurrentIssueIDs != `["`+concurrentIssue.Issue.Id+`"]` {
		t.Fatalf("concurrent legacy requirement references=%q err=%v", concurrentIssueIDs, err)
	}
	config := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if config.Code != http.StatusOK || !strings.Contains(config.Body.String(), `"issue_realtime":true`) {
		t.Fatalf("config=%d %s", config.Code, config.Body.String())
	}
}

func dialTokenRealtime(t *testing.T, serverURL, slug, token string) *websocket.Conn {
	t.Helper()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws?workspace_slug="+slug, http.Header{"Origin": []string{"http://localhost:3000"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteJSON(map[string]any{"type": "auth", "payload": map[string]string{"token": token}}); err != nil {
		t.Fatal(err)
	}
	assertExactRealtimeFrame(t, connection, `{"type":"auth_ack"}`)
	return connection
}

func assertTokenRealtimeDenied(t *testing.T, serverURL, slug, frame string) {
	t.Helper()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws?workspace_slug="+slug, http.Header{"Origin": []string{"http://localhost:3000"}})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
		t.Fatal(err)
	}
	assertExactRealtimeFrame(t, connection, `{"type":"auth_error","error":"authentication failed"}`)
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	if _, data, err := connection.ReadMessage(); err == nil {
		t.Fatalf("denied connection remained open: %s", data)
	}
}

func assertCookieRealtimeUpgradeDenied(t *testing.T, serverURL, slug, token string, status int) {
	t.Helper()
	header := http.Header{"Cookie": []string{"multica_auth=" + token}, "Origin": []string{"http://localhost:3000"}}
	connection, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws?workspace_slug="+slug, header)
	if err == nil {
		_ = connection.Close()
		t.Fatal("cookie upgrade unexpectedly accepted")
	}
	if response == nil || response.StatusCode != status {
		t.Fatalf("cookie denial=%#v %v", response, err)
	}
	body, _ := io.ReadAll(response.Body)
	if strings.TrimSpace(string(body)) != `{"error":"workspace not found"}` {
		t.Fatalf("cookie denial body=%s", body)
	}
}

func dialCookieRealtime(t *testing.T, serverURL, slug, token string) *websocket.Conn {
	t.Helper()
	header := http.Header{"Cookie": []string{"multica_auth=" + token}, "Origin": []string{"http://localhost:3000"}}
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws?workspace_slug="+slug, header)
	if err != nil {
		t.Fatal(err)
	}
	return connection
}

func assertRealtimeEvent(t *testing.T, connection *websocket.Conn, eventType, contains string) {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var message struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	if message.Type != eventType || (contains != "" && !strings.Contains(string(data), contains)) {
		t.Fatalf("event=%s want=%s contains=%s", data, eventType, contains)
	}
}

func assertExactRealtimeFrame(t *testing.T, connection *websocket.Conn, expected string) {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatalf("frame=%s want=%s", data, expected)
	}
}

func assertNoRealtimeEvent(t *testing.T, connection *websocket.Conn) {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(75 * time.Millisecond))
	if _, data, err := connection.ReadMessage(); err == nil {
		t.Fatalf("unexpected event: %s", data)
	}
	_ = connection.SetReadDeadline(time.Time{})
}
