package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeProjectSurfaceIDFailureDoesNotWriteOrHideMissingTargets(t *testing.T) {
	dependencies := FailClosedWorkspaceDependencies()
	dependencies.NewProjectID = func(context.Context) (string, error) { return "", errors.New("id unavailable") }
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: filepath.Join(t.TempDir(), "project-id-failure.db"), WorkspaceDependencies: dependencies,
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	owner := verifyRuntimeLogin(t, runtime, "project-id-failure@example.com")
	workspaceResponse := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Failure","slug":"failure"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	})
	var workspace struct {
		ID string `json:"id"`
	}
	if workspaceResponse.Code != http.StatusCreated || json.Unmarshal(workspaceResponse.Body.Bytes(), &workspace) != nil {
		t.Fatalf("create workspace = %d %s", workspaceResponse.Code, workspaceResponse.Body.String())
	}
	headers := map[string]string{"Authorization": "Bearer " + owner.Token, "X-Workspace-Slug": "failure", "Content-Type": "application/json"}
	failedProject := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"No ID"}`, headers)
	if failedProject.Code != http.StatusInternalServerError {
		t.Fatalf("failed project ID = %d %s", failedProject.Code, failedProject.Body.String())
	}
	var projectCount int
	if err := runtime.db.QueryRow(`SELECT COUNT(*) FROM workspace_projects WHERE workspace_id=?`, workspace.ID).Scan(&projectCount); err != nil || projectCount != 0 {
		t.Fatalf("failed project write count/error = %d/%v", projectCount, err)
	}
	missingPin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"missing"}`, headers)
	assertRuntimeResponse(t, missingPin.Code, missingPin.Body.String(), http.StatusNotFound, `{"error":"project not found"}`)
	if _, err := runtime.db.Exec(`INSERT INTO workspace_projects(id,workspace_id,name,description,status,asset_ids,created_at,updated_at) VALUES('seed-project',?,'Seed','','planned','[]','2026-08-14T00:00:00Z','2026-08-14T00:00:00Z')`, workspace.ID); err != nil {
		t.Fatal(err)
	}
	failedPin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"seed-project"}`, headers)
	if failedPin.Code != http.StatusInternalServerError {
		t.Fatalf("failed pin ID = %d %s", failedPin.Code, failedPin.Body.String())
	}
	var pinCount int
	if err := runtime.db.QueryRow(`SELECT COUNT(*) FROM workspace_pins`).Scan(&pinCount); err != nil || pinCount != 0 {
		t.Fatalf("failed pin write count/error = %d/%v", pinCount, err)
	}
}

func TestSQLiteRuntimeLoadsProjectsPageWithoutMissingRequests(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "projects-page.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "projects-owner@example.com")
	authHeaders := map[string]string{
		"Authorization": "Bearer " + login.Token,
		"Content-Type":  "application/json",
	}
	created := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Projects","slug":"projects"}`, authHeaders)
	if created.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", created.Code, created.Body.String())
	}
	var workspace struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{
		"Authorization":    "Bearer " + login.Token,
		"X-Workspace-Slug": "projects",
	}

	projects := runtimeRequest(runtime, http.MethodGet, "/api/projects?", "", headers)
	assertRuntimeResponse(t, projects.Code, projects.Body.String(), http.StatusOK, `{"projects":[],"total":0}`)

	members := runtimeRequest(runtime, http.MethodGet, "/api/workspaces/"+workspace.ID+"/members", "", headers)
	if members.Code != http.StatusOK {
		t.Fatalf("members = %d %s", members.Code, members.Body.String())
	}
	var memberRows []map[string]any
	if err := json.Unmarshal(members.Body.Bytes(), &memberRows); err != nil {
		t.Fatal(err)
	}
	if len(memberRows) != 1 || memberRows[0]["workspace_id"] != workspace.ID || memberRows[0]["user_id"] != login.UserID || memberRows[0]["role"] != "owner" {
		t.Fatalf("members = %s", members.Body.String())
	}

	pins := runtimeRequest(runtime, http.MethodGet, "/api/pins", "", headers)
	assertRuntimeResponse(t, pins.Code, pins.Body.String(), http.StatusOK, `[]`)
}

func TestSQLiteRuntimePersistsVisibleProjectAndPinActions(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "project-actions.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            databasePath,
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	owner := verifyRuntimeLogin(t, runtime, "project-actions@example.com")
	bearer := map[string]string{
		"Authorization":    "Bearer " + owner.Token,
		"X-Workspace-Slug": "delivery",
		"Content-Type":     "application/json",
	}
	createdWorkspace := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Delivery","slug":"delivery"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	})
	if createdWorkspace.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", createdWorkspace.Code, createdWorkspace.Body.String())
	}
	var workspace struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdWorkspace.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}

	cookieNoCSRF := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"Blocked"}`, map[string]string{
		"Cookie": "multica_auth=" + owner.Token, "X-Workspace-Slug": "delivery", "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, cookieNoCSRF.Code, cookieNoCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	unknown := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"Unknown","unknown":true}`, bearer)
	assertRuntimeResponse(t, unknown.Code, unknown.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)

	members := runtimeRequest(runtime, http.MethodGet, "/api/workspaces/"+workspace.ID+"/members", "", bearer)
	var memberRows []map[string]any
	if members.Code != http.StatusOK || json.Unmarshal(members.Body.Bytes(), &memberRows) != nil || len(memberRows) != 1 {
		t.Fatalf("members = %d %s", members.Code, members.Body.String())
	}
	leadUserID, _ := memberRows[0]["user_id"].(string)
	minimal := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"No Description"}`, bearer)
	if minimal.Code != http.StatusCreated || !strings.Contains(minimal.Body.String(), `"description":null`) {
		t.Fatalf("create project without description = %d %s", minimal.Code, minimal.Body.String())
	}
	var minimalProject map[string]any
	if err := json.Unmarshal(minimal.Body.Bytes(), &minimalProject); err != nil {
		t.Fatal(err)
	}
	minimalID, _ := minimalProject["id"].(string)
	minimalDelete := runtimeRequest(runtime, http.MethodDelete, "/api/projects/"+minimalID, "", bearer)
	assertRuntimeResponse(t, minimalDelete.Code, minimalDelete.Body.String(), http.StatusNoContent, ``)
	created := runtimeRequest(runtime, http.MethodPost, "/api/projects", `{"title":"Canonical Projects","description":"Visible project","icon":"rocket","status":"in_progress","priority":"high","lead_type":"member","lead_id":"`+leadUserID+`","start_date":"2026-08-14","due_date":"2026-08-30"}`, bearer)
	if created.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", created.Code, created.Body.String())
	}
	var project map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	projectID, _ := project["id"].(string)
	if projectID == "" || project["workspace_id"] != workspace.ID || project["title"] != "Canonical Projects" || project["priority"] != "high" || project["lead_id"] != leadUserID || project["issue_count"] != float64(0) {
		t.Fatalf("project = %s", created.Body.String())
	}
	createdIssue := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Project issue","project_id":"`+projectID+`"}`, bearer)
	if createdIssue.Code != http.StatusCreated {
		t.Fatalf("create project issue = %d %s", createdIssue.Code, createdIssue.Body.String())
	}
	var issue map[string]any
	if err := json.Unmarshal(createdIssue.Body.Bytes(), &issue); err != nil {
		t.Fatal(err)
	}
	if issue["id"] == "" || issue["workspace_id"] != workspace.ID || issue["project_id"] != projectID || issue["title"] != "Project issue" {
		t.Fatalf("created issue = %s", createdIssue.Body.String())
	}
	missingIssueAuth := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"missing auth"}`, map[string]string{
		"X-Workspace-Slug": "delivery", "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, missingIssueAuth.Code, missingIssueAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	missingIssueWorkspace := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"missing workspace"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, missingIssueWorkspace.Code, missingIssueWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)
	cookieIssueNoCSRF := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"missing csrf"}`, map[string]string{
		"Cookie": "multica_auth=" + owner.Token, "X-Workspace-Slug": "delivery", "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, cookieIssueNoCSRF.Code, cookieIssueNoCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	cookieIssue := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"Cookie issue","project_id":"`+projectID+`"}`, map[string]string{
		"Cookie": "multica_auth=" + owner.Token + "; multica_csrf=" + owner.CSRF, "X-CSRF-Token": owner.CSRF,
		"X-Workspace-Slug": "delivery", "Content-Type": "application/json",
	})
	if cookieIssue.Code != http.StatusCreated || !strings.Contains(cookieIssue.Body.String(), `"title":"Cookie issue"`) {
		t.Fatalf("cookie create issue = %d %s", cookieIssue.Code, cookieIssue.Body.String())
	}
	for name, body := range map[string]string{
		"unknown":             `{"title":"unknown","unknown":true}`,
		"trailing":            `{"title":"trailing"} {}`,
		"unsupported labels":  `{"title":"labels","label_ids":["label-one"]}`,
		"unsupported uploads": `{"title":"files","attachment_ids":["asset-one"]}`,
	} {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues", body, bearer)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s issue create = %d %s", name, response.Code, response.Body.String())
		}
	}
	oversizedIssue := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"`+strings.Repeat("x", (1<<20)+1)+`"}`, bearer)
	assertRuntimeResponse(t, oversizedIssue.Code, oversizedIssue.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)
	missingProjectIssue := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"hidden","project_id":"missing-project"}`, bearer)
	assertRuntimeResponse(t, missingProjectIssue.Code, missingProjectIssue.Body.String(), http.StatusNotFound, `{"error":"issue not found"}`)
	configResponse := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if configResponse.Code != http.StatusOK || !strings.Contains(configResponse.Body.String(), `"issue_create":true`) {
		t.Fatalf("issue create capability = %d %s", configResponse.Code, configResponse.Body.String())
	}

	listed := runtimeRequest(runtime, http.MethodGet, "/api/projects?status=in_progress", "", bearer)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"id":"`+projectID+`"`) {
		t.Fatalf("list projects = %d %s", listed.Code, listed.Body.String())
	}
	got := runtimeRequest(runtime, http.MethodGet, "/api/projects/"+projectID, "", bearer)
	if got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"title":"Canonical Projects"`) {
		t.Fatalf("get project = %d %s", got.Code, got.Body.String())
	}
	updated := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+projectID, `{"title":"Canonical Projects Updated","description":null,"status":"completed","priority":"urgent","icon":"  ","lead_id":" `+leadUserID+` ","due_date":""}`, bearer)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"title":"Canonical Projects Updated"`) || !strings.Contains(updated.Body.String(), `"description":null`) || !strings.Contains(updated.Body.String(), `"priority":"urgent"`) || !strings.Contains(updated.Body.String(), `"icon":null`) || !strings.Contains(updated.Body.String(), `"lead_id":"`+leadUserID+`"`) || !strings.Contains(updated.Body.String(), `"due_date":null`) {
		t.Fatalf("update project = %d %s", updated.Code, updated.Body.String())
	}
	invalidUpdate := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+projectID, `{"status":"invented"}`, bearer)
	assertRuntimeResponse(t, invalidUpdate.Code, invalidUpdate.Body.String(), http.StatusBadRequest, `{"error":"invalid project request"}`)
	trailingUpdate := runtimeRequest(runtime, http.MethodPut, "/api/projects/"+projectID, `{"title":"Ignored"} {}`, bearer)
	assertRuntimeResponse(t, trailingUpdate.Code, trailingUpdate.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)
	missingMalformedUpdate := runtimeRequest(runtime, http.MethodPut, "/api/projects/missing-project", `{"unknown":true}`, bearer)
	assertRuntimeResponse(t, missingMalformedUpdate.Code, missingMalformedUpdate.Body.String(), http.StatusNotFound, `{"error":"project not found"}`)
	missingTargetPin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"missing-project"}`, bearer)
	assertRuntimeResponse(t, missingTargetPin.Code, missingTargetPin.Body.String(), http.StatusNotFound, `{"error":"project not found"}`)
	if _, err := runtime.db.Exec(`INSERT INTO workspace_issues(
		id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,metadata,properties,asset_ids,created_at,updated_at
	) VALUES('project-surface-issue',?,99,'DEL-99','Pin issue','backlog','medium','member',?,'{}','{}','[]','2026-08-14T00:00:00Z','2026-08-14T00:00:00Z')`, workspace.ID, memberRows[0]["id"]); err != nil {
		t.Fatal(err)
	}
	issuePin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"issue","item_id":"project-surface-issue"}`, bearer)
	if issuePin.Code != http.StatusCreated || !strings.Contains(issuePin.Body.String(), `"item_type":"issue"`) {
		t.Fatalf("create issue pin = %d %s", issuePin.Code, issuePin.Body.String())
	}
	issueUnpin := runtimeRequest(runtime, http.MethodDelete, "/api/pins/issue/project-surface-issue", "", bearer)
	assertRuntimeResponse(t, issueUnpin.Code, issueUnpin.Body.String(), http.StatusNoContent, ``)

	pinned := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"`+projectID+`"}`, bearer)
	if pinned.Code != http.StatusCreated || !strings.Contains(pinned.Body.String(), `"item_id":"`+projectID+`"`) {
		t.Fatalf("create pin = %d %s", pinned.Code, pinned.Body.String())
	}
	duplicate := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"`+projectID+`"}`, bearer)
	assertRuntimeResponse(t, duplicate.Code, duplicate.Body.String(), http.StatusConflict, `{"error":"item already pinned"}`)
	missingAuth := runtimeRequest(runtime, http.MethodGet, "/api/projects", "", map[string]string{"X-Workspace-Slug": "delivery"})
	assertRuntimeResponse(t, missingAuth.Code, missingAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	missingWorkspace := runtimeRequest(runtime, http.MethodGet, "/api/projects", "", map[string]string{"Authorization": "Bearer " + owner.Token})
	assertRuntimeResponse(t, missingWorkspace.Code, missingWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)
	foreign := verifyRuntimeLogin(t, runtime, "project-outsider@example.com")
	foreignProject := runtimeRequest(runtime, http.MethodGet, "/api/projects/"+projectID, "", map[string]string{
		"Authorization": "Bearer " + foreign.Token, "X-Workspace-Slug": "delivery",
	})
	if foreignProject.Code != http.StatusNotFound {
		t.Fatalf("foreign project = %d %s", foreignProject.Code, foreignProject.Body.String())
	}
	foreignPin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"`+projectID+`"}`, map[string]string{
		"Authorization": "Bearer " + foreign.Token, "X-Workspace-Slug": "delivery", "Content-Type": "application/json",
	})
	if foreignPin.Code != http.StatusNotFound {
		t.Fatalf("foreign pin = %d %s", foreignPin.Code, foreignPin.Body.String())
	}
	if _, err := runtime.db.Exec(`UPDATE auth_sessions SET expires_at_unix_nano=0 WHERE user_id=?`, foreign.UserID); err != nil {
		t.Fatal(err)
	}
	expiredPins := runtimeRequest(runtime, http.MethodGet, "/api/pins", "", map[string]string{
		"Authorization": "Bearer " + foreign.Token, "X-Workspace-Slug": "delivery",
	})
	assertRuntimeResponse(t, expiredPins.Code, expiredPins.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	expiredIssueCreate := runtimeRequest(runtime, http.MethodPost, "/api/issues", `{"title":"expired"}`, map[string]string{
		"Authorization": "Bearer " + foreign.Token, "X-Workspace-Slug": "delivery", "Content-Type": "application/json",
	})
	assertRuntimeResponse(t, expiredIssueCreate.Code, expiredIssueCreate.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)
	pinCookieNoCSRF := runtimeRequest(runtime, http.MethodDelete, "/api/pins/project/"+projectID, "", map[string]string{
		"Cookie": "multica_auth=" + owner.Token, "X-Workspace-Slug": "delivery",
	})
	assertRuntimeResponse(t, pinCookieNoCSRF.Code, pinCookieNoCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	trailingPin := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"`+projectID+`"} {}`, bearer)
	assertRuntimeResponse(t, trailingPin.Code, trailingPin.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)

	member := verifyRuntimeLogin(t, runtime, "project-member@example.com")
	if _, err := runtime.db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('project-member-row',?,?, 'member','2026-08-14T00:00:00Z')`, workspace.ID, member.UserID); err != nil {
		t.Fatal(err)
	}
	memberDelete := runtimeRequest(runtime, http.MethodDelete, "/api/projects/"+projectID, "", map[string]string{
		"Authorization": "Bearer " + member.Token, "X-Workspace-Slug": "delivery",
	})
	assertRuntimeResponse(t, memberDelete.Code, memberDelete.Body.String(), http.StatusForbidden, `{"error":"insufficient workspace role"}`)
	memberMissingDelete := runtimeRequest(runtime, http.MethodDelete, "/api/projects/missing-project", "", map[string]string{
		"Authorization": "Bearer " + member.Token, "X-Workspace-Slug": "delivery",
	})
	assertRuntimeResponse(t, memberMissingDelete.Code, memberMissingDelete.Body.String(), http.StatusNotFound, `{"error":"project not found"}`)

	removedPin := runtimeRequest(runtime, http.MethodDelete, "/api/pins/project/"+projectID, "", bearer)
	assertRuntimeResponse(t, removedPin.Code, removedPin.Body.String(), http.StatusNoContent, ``)
	removedPinAgain := runtimeRequest(runtime, http.MethodDelete, "/api/pins/project/"+projectID, "", bearer)
	assertRuntimeResponse(t, removedPinAgain.Code, removedPinAgain.Body.String(), http.StatusNoContent, ``)
	start := make(chan struct{})
	codes := make(chan int, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodPost, "/api/pins", `{"item_type":"project","item_id":"`+projectID+`"}`, bearer)
			codes <- response.Code
		}()
	}
	close(start)
	workers.Wait()
	close(codes)
	createdPins, conflictedPins := 0, 0
	for code := range codes {
		switch code {
		case http.StatusCreated:
			createdPins++
		case http.StatusConflict:
			conflictedPins++
		default:
			t.Fatalf("concurrent pin status = %d", code)
		}
	}
	if createdPins != 1 || conflictedPins != 1 {
		t.Fatalf("concurrent pins created/conflicted = %d/%d", createdPins, conflictedPins)
	}

	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, config)
	retained := runtimeRequest(restarted, http.MethodGet, "/api/pins", "", bearer)
	if retained.Code != http.StatusOK || !strings.Contains(retained.Body.String(), `"item_id":"`+projectID+`"`) {
		t.Fatalf("restart pins = %d %s", retained.Code, retained.Body.String())
	}
	restartedProject := runtimeRequest(restarted, http.MethodGet, "/api/projects/"+projectID, "", bearer)
	if restartedProject.Code != http.StatusOK || !strings.Contains(restartedProject.Body.String(), `"title":"Canonical Projects Updated"`) {
		t.Fatalf("restart project = %d %s", restartedProject.Code, restartedProject.Body.String())
	}
	if _, err := restarted.db.Exec(`CREATE TRIGGER reject_project_surface_delete BEFORE DELETE ON workspace_projects WHEN OLD.id = '` + projectID + `' BEGIN SELECT RAISE(ABORT, 'reject project delete'); END`); err != nil {
		t.Fatal(err)
	}
	rolledBack := runtimeRequest(restarted, http.MethodDelete, "/api/projects/"+projectID, "", bearer)
	if rolledBack.Code != http.StatusInternalServerError {
		t.Fatalf("forced rollback delete = %d %s", rolledBack.Code, rolledBack.Body.String())
	}
	retainedAfterRollback := runtimeRequest(restarted, http.MethodGet, "/api/pins", "", bearer)
	if retainedAfterRollback.Code != http.StatusOK || !strings.Contains(retainedAfterRollback.Body.String(), `"item_id":"`+projectID+`"`) {
		t.Fatalf("rollback retained pin = %d %s", retainedAfterRollback.Code, retainedAfterRollback.Body.String())
	}
	if _, err := restarted.db.Exec(`DROP TRIGGER reject_project_surface_delete`); err != nil {
		t.Fatal(err)
	}

	deleted := runtimeRequest(restarted, http.MethodDelete, "/api/projects/"+projectID, "", bearer)
	assertRuntimeResponse(t, deleted.Code, deleted.Body.String(), http.StatusNoContent, ``)
	remainingPins := runtimeRequest(restarted, http.MethodGet, "/api/pins", "", bearer)
	assertRuntimeResponse(t, remainingPins.Code, remainingPins.Body.String(), http.StatusOK, `[]`)
	missingProject := runtimeRequest(restarted, http.MethodGet, "/api/projects/"+projectID, "", bearer)
	assertRuntimeResponse(t, missingProject.Code, missingProject.Body.String(), http.StatusNotFound, `{"error":"project not found"}`)
}
