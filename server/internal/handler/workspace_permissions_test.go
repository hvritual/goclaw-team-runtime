package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/workspacepermissions"
)

func TestWorkspacePermissionManagementIsAdminOnly(t *testing.T) {
	ctx := context.Background()
	email := "permissions-member-" + uuid.NewString() + "@multica.test"
	var memberUserID, memberID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO "user" (name, email) VALUES ('Permission Member', $1) RETURNING id
	`, email).Scan(&memberUserID); err != nil {
		t.Fatalf("create permission member user: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO member (workspace_id, user_id, role)
		VALUES ($1, $2, 'member') RETURNING id
	`, testWorkspaceID, memberUserID).Scan(&memberID); err != nil {
		t.Fatalf("create permission member: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM member WHERE id = $1`, memberID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, memberUserID)
	})

	ownerRecorder := httptest.NewRecorder()
	ownerRequest := withURLParam(
		newRequestAs(testUserID, http.MethodGet, "/api/workspaces/"+testWorkspaceID+"/permissions", nil),
		"id",
		testWorkspaceID,
	)
	testHandler.GetWorkspacePermissions(ownerRecorder, ownerRequest)
	if ownerRecorder.Code != http.StatusOK {
		t.Fatalf("owner permission catalog: got %d: %s", ownerRecorder.Code, ownerRecorder.Body.String())
	}
	var catalog workspacepermissions.Catalog
	if err := json.NewDecoder(ownerRecorder.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode permission catalog: %v", err)
	}
	if len(catalog.Roles) != 3 || len(catalog.Capabilities) == 0 {
		t.Fatalf("unexpected permission catalog: %#v", catalog)
	}

	memberRecorder := httptest.NewRecorder()
	memberRequest := withURLParam(
		newRequestAs(memberUserID, http.MethodGet, "/api/workspaces/"+testWorkspaceID+"/permissions", nil),
		"id",
		testWorkspaceID,
	)
	testHandler.GetWorkspacePermissions(memberRecorder, memberRequest)
	if memberRecorder.Code != http.StatusForbidden {
		t.Fatalf("member permission catalog: got %d, want 403: %s", memberRecorder.Code, memberRecorder.Body.String())
	}

	invitationRecorder := httptest.NewRecorder()
	invitationRequest := withURLParam(
		newRequestAs(memberUserID, http.MethodGet, "/api/workspaces/"+testWorkspaceID+"/invitations", nil),
		"id",
		testWorkspaceID,
	)
	testHandler.ListWorkspaceInvitations(invitationRecorder, invitationRequest)
	if invitationRecorder.Code != http.StatusForbidden {
		t.Fatalf("member invitation list: got %d, want 403: %s", invitationRecorder.Code, invitationRecorder.Body.String())
	}
}

func TestPostgresWorkspaceOwnerManagementProtectsLastOwner(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.NewString()
	var workspaceID, memberID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO workspace (name, slug, description, issue_prefix)
		VALUES ('Owner invariant', $1, '', 'OWN')
		RETURNING id
	`, "owner-invariant-"+suffix).Scan(&workspaceID); err != nil {
		t.Fatalf("create owner-invariant workspace: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO member (workspace_id, user_id, role)
		VALUES ($1, $2, 'owner')
		RETURNING id
	`, workspaceID, testUserID).Scan(&memberID); err != nil {
		t.Fatalf("create owner membership: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM member WHERE id = $1`, memberID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE id = $1`, workspaceID)
	})

	updateRecorder := httptest.NewRecorder()
	updateRequest := withURLParams(
		newRequestAs(
			testUserID,
			http.MethodPatch,
			"/api/workspaces/"+workspaceID+"/members/"+memberID,
			map[string]any{"role": "admin"},
		),
		"id",
		workspaceID,
		"memberId",
		memberID,
	)
	testHandler.UpdateMember(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("demote last owner: got %d, want 400: %s", updateRecorder.Code, updateRecorder.Body.String())
	}

	deleteRecorder := httptest.NewRecorder()
	deleteRequest := withURLParams(
		newRequestAs(
			testUserID,
			http.MethodDelete,
			"/api/workspaces/"+workspaceID+"/members/"+memberID,
			nil,
		),
		"id",
		workspaceID,
		"memberId",
		memberID,
	)
	testHandler.DeleteMember(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusBadRequest {
		t.Fatalf("remove last owner: got %d, want 400: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	leaveRecorder := httptest.NewRecorder()
	leaveRequest := withURLParams(
		newRequestAs(
			testUserID,
			http.MethodDelete,
			"/api/workspaces/"+workspaceID+"/leave",
			nil,
		),
		"id",
		workspaceID,
	)
	testHandler.LeaveWorkspace(leaveRecorder, leaveRequest)
	if leaveRecorder.Code != http.StatusBadRequest {
		t.Fatalf("last owner leaves: got %d, want 400: %s", leaveRecorder.Code, leaveRecorder.Body.String())
	}

	var role string
	if err := testPool.QueryRow(ctx, `SELECT role FROM member WHERE id = $1`, memberID).Scan(&role); err != nil {
		t.Fatalf("load owner after rejected operations: %v", err)
	}
	if role != "owner" {
		t.Fatalf("last owner role changed after rejected operations: got %q", role)
	}
}

func TestPostgresConcurrentOwnerDemotionsPreserveAnOwner(t *testing.T) {
	ctx := context.Background()
	suffix := uuid.NewString()
	email := "concurrent-owner-" + suffix + "@multica.test"

	var secondUserID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO "user" (name, email)
		VALUES ('Concurrent Owner', $1)
		RETURNING id
	`, email).Scan(&secondUserID); err != nil {
		t.Fatalf("create second owner user: %v", err)
	}

	var workspaceID, firstMemberID, secondMemberID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO workspace (name, slug, description, issue_prefix)
		VALUES ('Concurrent owner invariant', $1, '', 'OWN')
		RETURNING id
	`, "concurrent-owner-invariant-"+suffix).Scan(&workspaceID); err != nil {
		t.Fatalf("create concurrent owner workspace: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO member (workspace_id, user_id, role)
		VALUES ($1, $2, 'owner')
		RETURNING id
	`, workspaceID, testUserID).Scan(&firstMemberID); err != nil {
		t.Fatalf("create first owner membership: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		INSERT INTO member (workspace_id, user_id, role)
		VALUES ($1, $2, 'owner')
		RETURNING id
	`, workspaceID, secondUserID).Scan(&secondMemberID); err != nil {
		t.Fatalf("create second owner membership: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM member WHERE workspace_id = $1`, workspaceID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE id = $1`, workspaceID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, secondUserID)
	})

	type demotion struct {
		userID   string
		memberID string
	}
	demotions := []demotion{
		{userID: testUserID, memberID: firstMemberID},
		{userID: secondUserID, memberID: secondMemberID},
	}
	start := make(chan struct{})
	statuses := make(chan int, len(demotions))
	var waitGroup sync.WaitGroup
	for _, item := range demotions {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			recorder := httptest.NewRecorder()
			request := withURLParams(
				newRequestAs(
					item.userID,
					http.MethodPatch,
					"/api/workspaces/"+workspaceID+"/members/"+item.memberID,
					map[string]any{"role": "admin"},
				),
				"id",
				workspaceID,
				"memberId",
				item.memberID,
			)
			testHandler.UpdateMember(recorder, request)
			statuses <- recorder.Code
		}()
	}
	close(start)
	waitGroup.Wait()
	close(statuses)

	statusCounts := map[int]int{}
	for status := range statuses {
		statusCounts[status]++
	}
	if statusCounts[http.StatusOK] != 1 || statusCounts[http.StatusBadRequest] != 1 {
		t.Fatalf("concurrent demotions returned unexpected statuses: %#v", statusCounts)
	}

	var ownerCount int
	if err := testPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM member
		WHERE workspace_id = $1 AND role = 'owner'
	`, workspaceID).Scan(&ownerCount); err != nil {
		t.Fatalf("count owners after concurrent demotions: %v", err)
	}
	if ownerCount != 1 {
		t.Fatalf("concurrent demotions left %d owners, want 1", ownerCount)
	}
}
