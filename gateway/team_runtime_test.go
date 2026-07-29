package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

type gatewayTeamFixture struct {
	service teamcontrol.Service
	alice   teamcontrol.User
	bob     teamcontrol.User
	viewer  teamcontrol.User
	team    teamcontrol.Team
	project teamcontrol.Project
	other   teamcontrol.Project
	repo    teamcontrol.Repository
	policy  teamcontrol.ResolvedPolicy
	repoDir string
}

func newGatewayTeamFixture(t *testing.T) gatewayTeamFixture {
	t.Helper()
	service, err := teamcontrol.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	alice := mustCreateUser(t, service, "alice")
	bob := mustCreateUser(t, service, "bob")
	viewer := mustCreateUser(t, service, "viewer")
	team, err := service.CreateTeam(alice.ID, teamcontrol.CreateTeamInput{
		ID: "team-iot", Name: "IoT",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []teamcontrol.User{bob, viewer} {
		if _, err := service.AddTeamMember(
			alice.ID,
			team.ID,
			teamcontrol.AddTeamMemberInput{
				UserID: user.ID,
				Role:   teamcontrol.TeamRegularMember,
			},
		); err != nil {
			t.Fatal(err)
		}
	}
	project, err := service.CreateProject(alice.ID, teamcontrol.CreateProjectInput{
		ID: "project-alpha", TeamID: team.ID, Key: "alpha", Name: "Alpha",
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.CreateProject(alice.ID, teamcontrol.CreateProjectInput{
		ID: "project-other", TeamID: team.ID, Key: "other", Name: "Other",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddProjectMember(
		alice.ID,
		project.ID,
		teamcontrol.AddProjectMemberInput{
			UserID: bob.ID, Role: teamcontrol.ProjectDeveloper,
		},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddProjectMember(
		alice.ID,
		other.ID,
		teamcontrol.AddProjectMemberInput{
			UserID: viewer.ID, Role: teamcontrol.ProjectViewer,
		},
	); err != nil {
		t.Fatal(err)
	}
	repoDir := initGatewayGitRepository(t)
	repository, err := service.CreateRepository(
		alice.ID,
		teamcontrol.CreateRepositoryInput{
			ID:            "repo-alpha",
			ProjectID:     project.ID,
			Name:          "alpha",
			RemoteURL:     "https://example.invalid/alpha.git",
			LocalPath:     repoDir,
			DefaultBranch: "main",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PutPolicyBundle(
		alice.ID,
		teamcontrol.PutPolicyBundleInput{
			ID:      "policy-alpha",
			Name:    "engineering",
			Scope:   teamcontrol.PolicyProject,
			ScopeID: project.ID,
			Version: 1,
			Enabled: true,
			Rules: map[string]json.RawMessage{
				"style": json.RawMessage(`"gofmt"`),
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	policy, err := service.ResolvePolicy(
		alice.ID,
		project.ID,
		repository.ID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	return gatewayTeamFixture{
		service: *service,
		alice:   alice,
		bob:     bob,
		viewer:  viewer,
		team:    team,
		project: project,
		other:   other,
		repo:    repository,
		policy:  policy,
		repoDir: repoDir,
	}
}

func TestTeamPersonalTokenAuthenticationLayers(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	token := "alice-personal-token-123456789"
	if _, err := fixture.service.RegisterAccessToken(
		fixture.alice.ID,
		fixture.alice.ID,
		"test",
		token,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	server := &Server{
		teamSvc:    &fixture.service,
		enableAuth: true,
		authToken:  "gateway-outer-token",
	}

	request := newRPCRequestWithHeaders(token)
	userID, browserSession, err := server.authenticateTeamHTTP(request)
	if err != nil || userID != fixture.alice.ID || browserSession != nil {
		t.Fatalf(
			"HTTP personal token authentication failed: user=%q browser_session=%v err=%v",
			userID,
			browserSession != nil,
			err,
		)
	}
	request.Header.Del("X-GoClaw-User-Token")
	if _, _, err := server.authenticateTeamHTTP(request); err == nil {
		t.Fatal("team HTTP authentication accepted a missing personal token")
	}

	wsRequest := newRPCRequestWithHeaders("")
	encoded := base64.RawURLEncoding.EncodeToString([]byte(token))
	wsRequest.Header.Set(
		"Sec-WebSocket-Protocol",
		"goclaw.v1, goclaw.user."+encoded,
	)
	userID, err = server.authenticateTeamWebSocket(wsRequest)
	if err != nil || userID != fixture.alice.ID {
		t.Fatalf("WebSocket personal token authentication failed: user=%q err=%v", userID, err)
	}
}

func TestControlPlaneRegistryRPCsAreProjectScoped(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	session := teamSessionID(fixture.alice.ID)
	checksum := strings.Repeat("a", 64)

	if _, err := handler.registry.Call(
		"budget.put",
		session,
		map[string]interface{}{
			"id": "budget-rpc", "project_id": fixture.project.ID,
			"user_id": fixture.bob.ID, "limit_tokens": 500,
		},
	); err != nil {
		t.Fatal(err)
	}
	knowledgeResult, err := handler.registry.Call(
		"knowledge.source.put",
		session,
		map[string]interface{}{
			"id": "knowledge-rpc", "project_id": fixture.project.ID,
			"name": "RPC knowledge", "uri": "file:///vault/rpc.md",
			"revision": "1", "sha256": checksum, "status": "approved",
			"metadata": map[string]string{"source_kind": "documentation"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	knowledgeJSON, err := json.Marshal(knowledgeResult)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(knowledgeJSON), "metadata") {
		t.Fatalf("registry metadata escaped Gateway presenter: %s", knowledgeJSON)
	}
	if _, err := handler.registry.Call(
		"skill.release.put",
		session,
		map[string]interface{}{
			"id": "skill-rpc", "project_id": fixture.project.ID,
			"name": "rpc-skill", "version": "1", "uri": "file:///skills/rpc",
			"sha256": checksum, "status": "approved",
		},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"runner.release.put",
		session,
		map[string]interface{}{
			"id": "runner-rpc", "project_id": fixture.project.ID,
			"channel": "pilot", "version": "1", "os": "linux", "arch": "amd64",
			"uri":    "https://example.invalid/runner.tar.gz",
			"sha256": checksum, "size_bytes": 1024,
			"min_protocol": "1", "status": "approved",
		},
	); err != nil {
		t.Fatal(err)
	}
	compiled, err := handler.registry.Call(
		"context.compile",
		session,
		map[string]interface{}{
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"user_id":       fixture.bob.ID,
			"budget_id":     "budget-rpc",
			"knowledge_ids": []string{"knowledge-rpc"},
			"skill_ids":     []string{"skill-rpc"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	bundle, ok := compiled.(teamcontrol.ContextBundle)
	if !ok || bundle.Hash == "" {
		t.Fatalf("unexpected context bundle: %#v", compiled)
	}
	summary, err := handler.registry.Call(
		"control.summary",
		session,
		map[string]interface{}{"project_id": fixture.project.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	summaryMap, ok := summary.(map[string]interface{})
	if !ok || summaryMap["context_bundle_count"] != 1 {
		t.Fatalf("unexpected control summary: %#v", summary)
	}

	if _, err := handler.registry.Call(
		"budget.list",
		teamSessionID(fixture.viewer.ID),
		map[string]interface{}{"project_id": fixture.project.ID},
	); err == nil {
		t.Fatal("cross-project viewer read was allowed")
	}
	if _, err := handler.registry.Call(
		"budget.put",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id": "developer-budget", "project_id": fixture.project.ID,
			"limit_tokens": 1,
		},
	); err == nil {
		t.Fatal("developer budget mutation was allowed")
	}

	if _, err := handler.registry.Call(
		"knowledge.source.get",
		session,
		map[string]interface{}{
			"project_id": fixture.project.ID, "knowledge_id": "knowledge-rpc",
		},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"knowledge.source.delete",
		session,
		map[string]interface{}{
			"project_id": fixture.project.ID, "knowledge_id": "knowledge-rpc",
		},
	); !errors.Is(err, teamcontrol.ErrConflict) {
		t.Fatalf("approved registry delete error = %v, want conflict", err)
	}
	if _, err := handler.registry.Call(
		"knowledge.source.put",
		session,
		map[string]interface{}{
			"id": "knowledge-rpc", "project_id": fixture.project.ID,
			"name": "RPC knowledge", "uri": "file:///vault/rpc.md",
			"revision": "1", "sha256": checksum, "status": "disabled",
			"metadata": map[string]string{"source_kind": "documentation"},
		},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"knowledge.source.delete",
		session,
		map[string]interface{}{
			"project_id": fixture.project.ID, "knowledge_id": "knowledge-rpc",
		},
	); err != nil {
		t.Fatal(err)
	}

	for _, registry := range []struct {
		getMethod    string
		putMethod    string
		deleteMethod string
		idParam      string
		id           string
		put          map[string]interface{}
	}{
		{
			getMethod: "skill.release.get", putMethod: "skill.release.put",
			deleteMethod: "skill.release.delete", idParam: "skill_id", id: "skill-rpc",
			put: map[string]interface{}{
				"id": "skill-rpc", "project_id": fixture.project.ID,
				"name": "rpc-skill", "version": "1", "uri": "file:///skills/rpc",
				"sha256": checksum, "status": "disabled",
			},
		},
		{
			getMethod: "runner.release.get", putMethod: "runner.release.put",
			deleteMethod: "runner.release.delete", idParam: "runner_release_id",
			id: "runner-rpc",
			put: map[string]interface{}{
				"id": "runner-rpc", "project_id": fixture.project.ID,
				"channel": "pilot", "version": "1", "os": "linux", "arch": "amd64",
				"uri":    "https://example.invalid/runner.tar.gz",
				"sha256": checksum, "size_bytes": 1024,
				"min_protocol": "1", "status": "disabled",
			},
		},
	} {
		params := map[string]interface{}{
			"project_id": fixture.project.ID, registry.idParam: registry.id,
		}
		if _, err := handler.registry.Call(registry.getMethod, session, params); err != nil {
			t.Fatal(err)
		}
		if _, err := handler.registry.Call(
			registry.deleteMethod, session, params,
		); !errors.Is(err, teamcontrol.ErrConflict) {
			t.Fatalf("%s approved delete error = %v, want conflict", registry.id, err)
		}
		if _, err := handler.registry.Call(
			registry.getMethod,
			teamSessionID(fixture.viewer.ID),
			params,
		); !errors.Is(err, teamcontrol.ErrForbidden) {
			t.Fatalf("%s cross-project read error = %v, want forbidden", registry.id, err)
		}
		otherParams := map[string]interface{}{
			"project_id": fixture.other.ID, registry.idParam: registry.id,
		}
		if _, err := handler.registry.Call(
			registry.getMethod, session, otherParams,
		); !errors.Is(err, teamcontrol.ErrNotFound) {
			t.Fatalf("%s wrong-project read error = %v, want not found", registry.id, err)
		}
		if _, err := handler.registry.Call(registry.putMethod, session, registry.put); err != nil {
			t.Fatal(err)
		}
		if _, err := handler.registry.Call(registry.deleteMethod, session, params); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLegacyUnsafeUsageMetadataDoesNotEscapeRPC(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	budget, err := fixture.service.PutTokenBudget(
		fixture.alice.ID,
		teamcontrol.PutTokenBudgetInput{
			ID: "legacy-rpc-budget", ProjectID: fixture.project.ID,
			LimitTokens: 10,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.RecordTokenUsage(
		fixture.alice.ID,
		teamcontrol.RecordTokenUsageInput{
			ID: "legacy-rpc-usage", ProjectID: fixture.project.ID,
			BudgetID: budget.ID, Tokens: 1,
		},
	); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(fixture.service.Config().Root, "teamcontrol.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	var events map[string]map[string]interface{}
	if err := json.Unmarshal(document["token_usage_events"], &events); err != nil {
		t.Fatal(err)
	}
	for key, event := range events {
		event["metadata"] = map[string]string{"provider_token": "synthetic-legacy"}
		events[key] = event
	}
	document["token_usage_events"], err = json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	reopened, err := teamcontrol.Open(statePath)
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(reopened)
	_, err = handler.registry.Call(
		"budget.usage.list",
		teamSessionID(fixture.alice.ID),
		map[string]interface{}{"project_id": fixture.project.ID},
	)
	if err == nil || !strings.Contains(err.Error(), "metadata schema validation") {
		t.Fatalf("legacy usage metadata RPC error = %v, want fail-closed", err)
	}
}

func TestTeamTokenRPCDoesNotExposeCredentialDigest(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)

	issued, err := handler.registry.Call(
		"team.token.issue",
		teamSessionID(fixture.alice.ID),
		map[string]interface{}{
			"user_id": fixture.bob.ID,
			"label":   "bob-laptop",
			"token":   "bob-personal-token-123456789",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertNoCredentialDigest(t, issued)

	listed, err := handler.registry.Call(
		"team.token.list",
		teamSessionID(fixture.alice.ID),
		map[string]interface{}{"user_id": fixture.bob.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertNoCredentialDigest(t, listed)
}

func assertNoCredentialDigest(t *testing.T, value interface{}) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "token_sha256") {
		t.Fatalf("credential digest leaked through Gateway: %s", data)
	}
}

func TestTeamChatUsesSharedProjectRouteAndIsolation(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	handler := &Handler{teamSvc: &fixture.service}
	_, aliceChat, _, err := handler.chatIdentity(
		teamSessionID(fixture.alice.ID),
		map[string]interface{}{"project_id": fixture.project.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, bobChat, _, err := handler.chatIdentity(
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{"project_id": fixture.project.ID, "topic_id": "inbox"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if aliceChat != bobChat ||
		aliceChat != "project:project-alpha:topic:inbox" {
		t.Fatalf("team members did not share stable chat route: %q %q", aliceChat, bobChat)
	}
	if got := gatewayProjectID(aliceChat, nil); got != fixture.project.ID {
		t.Fatalf("project route resolved to %q", got)
	}
	if got := gatewayTopicID(aliceChat, nil); got != "inbox" {
		t.Fatalf("topic route resolved to %q", got)
	}
	server := &Server{teamSvc: &fixture.service}
	if !server.connectionCanReadProject(
		&Connection{PrincipalID: fixture.alice.ID},
		fixture.project.ID,
	) || !server.connectionCanReadProject(
		&Connection{PrincipalID: fixture.bob.ID},
		fixture.project.ID,
	) {
		t.Fatal("same-project members should receive project events")
	}
	if server.connectionCanReadProject(
		&Connection{PrincipalID: fixture.viewer.ID},
		fixture.project.ID,
	) {
		t.Fatal("cross-project member received project event")
	}
}

func TestTeamProjectionUsesCanonicalLifecycleStatus(t *testing.T) {
	issueStatuses := []teamcontrol.IssueStatus{
		teamcontrol.IssueNew,
		teamcontrol.IssueTriaged,
		teamcontrol.IssueAssigned,
		teamcontrol.IssueInProgress,
		teamcontrol.IssueBlocked,
		teamcontrol.IssueVerifying,
		teamcontrol.IssueResolved,
		teamcontrol.IssueClosed,
		teamcontrol.IssueReopened,
		teamcontrol.IssueCancelled,
	}
	for _, status := range issueStatuses {
		t.Run("issue-"+string(status), func(t *testing.T) {
			projected := presentIssue(teamcontrol.Issue{
				ID:     "issue-projection",
				Status: status,
			}, "alice")
			if projected["status"] != string(status) ||
				projected["lifecycle_status"] != string(status) ||
				projected["owner_id"] != "alice" {
				t.Fatalf("issue projection collapsed lifecycle state: %+v", projected)
			}
		})
	}

	workItemStatuses := []teamcontrol.WorkItemStatus{
		teamcontrol.WorkItemPending,
		teamcontrol.WorkItemReady,
		teamcontrol.WorkItemInProgress,
		teamcontrol.WorkItemBlocked,
		teamcontrol.WorkItemVerifying,
		teamcontrol.WorkItemDone,
		teamcontrol.WorkItemCancelled,
	}
	for _, status := range workItemStatuses {
		t.Run("work-"+string(status), func(t *testing.T) {
			projected := presentWorkItem(teamcontrol.WorkItem{
				ID:     "work-projection",
				Status: status,
			}, "bob")
			if projected["status"] != string(status) ||
				projected["lifecycle_status"] != string(status) ||
				projected["assignee_id"] != "bob" {
				t.Fatalf("work projection collapsed lifecycle state: %+v", projected)
			}
		})
	}
}

func TestTeamLinkedTerminalGuardAndTaskProjection(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	issue, err := fixture.service.CreateIssue(
		fixture.bob.ID,
		teamcontrol.CreateIssueInput{
			ID:        "issue-terminal-guard",
			ProjectID: fixture.project.ID,
			Type:      teamcontrol.IssueBug,
			Title:     "Terminal state must follow DoneGate",
			Severity:  teamcontrol.SeverityHigh,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	issue, err = fixture.service.TransitionIssue(
		fixture.bob.ID,
		fixture.project.ID,
		issue.ID,
		teamcontrol.IssueTriaged,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Assign(
		fixture.bob.ID,
		teamcontrol.AssignInput{
			ID:         "assign-terminal-issue",
			ProjectID:  fixture.project.ID,
			TargetType: teamcontrol.AssignmentIssue,
			TargetID:   issue.ID,
			UserID:     fixture.bob.ID,
			Role:       teamcontrol.AssignmentOwner,
		},
	); err != nil {
		t.Fatal(err)
	}
	issue, err = fixture.service.TransitionIssue(
		fixture.bob.ID,
		fixture.project.ID,
		issue.ID,
		teamcontrol.IssueInProgress,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	issue, err = fixture.service.TransitionIssue(
		fixture.bob.ID,
		fixture.project.ID,
		issue.ID,
		teamcontrol.IssueVerifying,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	workItem, err := fixture.service.CreateWorkItem(
		fixture.bob.ID,
		teamcontrol.CreateWorkItemInput{
			ID:           "work-terminal-guard",
			ProjectID:    fixture.project.ID,
			IssueID:      issue.ID,
			Title:        "Guard terminal state",
			Instructions: "Require the linked development lifecycle.",
			Priority:     teamcontrol.PriorityP1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Assign(
		fixture.bob.ID,
		teamcontrol.AssignInput{
			ID:         "assign-terminal-work",
			ProjectID:  fixture.project.ID,
			TargetType: teamcontrol.AssignmentWorkItem,
			TargetID:   workItem.ID,
			UserID:     fixture.bob.ID,
			Role:       teamcontrol.AssignmentOwner,
		},
	); err != nil {
		t.Fatal(err)
	}
	for _, status := range []teamcontrol.WorkItemStatus{
		teamcontrol.WorkItemInProgress,
		teamcontrol.WorkItemVerifying,
	} {
		workItem, err = fixture.service.TransitionWorkItem(
			fixture.bob.ID,
			fixture.project.ID,
			workItem.ID,
			status,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	development, err := dev.NewService(dev.Config{
		Root:     t.TempDir(),
		RepoPath: fixture.repoDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := development.CreateTask(dev.CreateRequest{
		ID:           "task-terminal-guard",
		ProjectID:    fixture.project.ID,
		RepositoryID: fixture.repo.ID,
		AssigneeID:   fixture.bob.ID,
		IssueIDs:     []string{issue.ID},
		Title:        "Terminal guard",
		RepoPath:     fixture.repoDir,
		Request:      dev.RequestFrame{RawRequest: "Guard terminal transitions."},
		Plan: dev.PlanSpec{Milestones: []dev.Milestone{{
			ID: "terminal", Title: "Terminal", WorkItems: []dev.WorkItem{{
				ID:           workItem.ID,
				Title:        workItem.Title,
				Instructions: workItem.Instructions,
			}},
		}}},
		CreatedBy: teamcontrol.PlannerServicePrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetDevelopmentService(development)

	issuesResult, err := handler.registry.Call(
		"issue.list",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{"project_id": fixture.project.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectedTaskID(t, issuesResult, issue.ID, task.ID)
	workResult, err := handler.registry.Call(
		"work.items",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{"project_id": fixture.project.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectedTaskID(t, workResult, workItem.ID, task.ID)

	for _, attempt := range []struct {
		method string
		params map[string]interface{}
	}{
		{
			method: "issue.transition",
			params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"issue_id":   issue.ID,
				"status":     string(teamcontrol.IssueResolved),
				"resolution": "must not bypass the task",
			},
		},
		{
			method: "work.transition",
			params: map[string]interface{}{
				"project_id":   fixture.project.ID,
				"work_item_id": workItem.ID,
				"status":       string(teamcontrol.WorkItemDone),
			},
		},
	} {
		if _, err := handler.registry.Call(
			attempt.method,
			teamSessionID(fixture.bob.ID),
			attempt.params,
		); err == nil || !strings.Contains(err.Error(), "linked development task") {
			t.Fatalf("%s bypassed the linked task DoneGate: %v", attempt.method, err)
		}
	}

	if _, err := development.CancelTask(
		task.ID,
		fixture.bob.ID,
		"Cancel the linked pilot task before cancelling its resources.",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"issue.transition",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"project_id": fixture.project.ID,
			"issue_id":   issue.ID,
			"status":     string(teamcontrol.IssueBlocked),
		},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"work.transition",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"project_id":   fixture.project.ID,
			"work_item_id": workItem.ID,
			"status":       string(teamcontrol.WorkItemBlocked),
		},
	); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		method string
		params map[string]interface{}
	}{
		{
			method: "issue.transition",
			params: map[string]interface{}{
				"project_id": fixture.project.ID,
				"issue_id":   issue.ID,
				"status":     string(teamcontrol.IssueCancelled),
			},
		},
		{
			method: "work.transition",
			params: map[string]interface{}{
				"project_id":   fixture.project.ID,
				"work_item_id": workItem.ID,
				"status":       string(teamcontrol.WorkItemCancelled),
			},
		},
	} {
		if _, err := handler.registry.Call(
			attempt.method,
			teamSessionID(fixture.bob.ID),
			attempt.params,
		); err != nil {
			t.Fatalf("%s rejected a terminal state after task cancellation: %v", attempt.method, err)
		}
	}
}

func assertProjectedTaskID(
	t *testing.T,
	result interface{},
	resourceID string,
	taskID string,
) {
	t.Helper()
	items, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatalf("unexpected projection type %T", result)
	}
	for _, item := range items {
		if item["id"] == resourceID {
			if item["task_id"] != taskID {
				t.Fatalf("resource %q task_id = %#v", resourceID, item["task_id"])
			}
			if item["status"] != item["lifecycle_status"] {
				t.Fatalf("resource %q lifecycle projection diverged: %+v", resourceID, item)
			}
			return
		}
	}
	t.Fatalf("resource %q not found in projection: %+v", resourceID, items)
}

func TestTeamDevCreateBindsCreatorAndValidatesAssigneeAndRepository(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	development, err := dev.NewService(dev.Config{
		Root:                  t.TempDir(),
		RepoPath:              fixture.repoDir,
		GatewayAllowExecution: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	workItem, err := fixture.service.CreateWorkItem(
		fixture.bob.ID,
		teamcontrol.CreateWorkItemInput{
			ID:           "work-server-authority",
			ProjectID:    fixture.project.ID,
			Title:        "Server authority",
			Instructions: "Verify server-owned task identity.",
			Priority:     teamcontrol.PriorityP2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Assign(
		fixture.bob.ID,
		teamcontrol.AssignInput{
			ID:         "assign-server-authority",
			ProjectID:  fixture.project.ID,
			TargetType: teamcontrol.AssignmentWorkItem,
			TargetID:   workItem.ID,
			UserID:     fixture.bob.ID,
			Role:       teamcontrol.AssignmentOwner,
		},
	); err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetDevelopmentService(development)

	if _, err := handler.registry.Call(
		"dev.task.create",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":            "task-missing-wave",
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"title":         "Missing Wave",
			"request": map[string]interface{}{
				"raw_request": "This team task must fail closed.",
			},
		},
	); err == nil || !strings.Contains(err.Error(), "wave binding is required") {
		t.Fatalf("team task without Wave binding did not fail closed: %v", err)
	}
	if _, err := handler.registry.Call(
		"dev.task.create",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":            "task-missing-wave-step",
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"wave":          map[string]interface{}{},
			"title":         "Missing Wave step",
			"request": map[string]interface{}{
				"raw_request": "This team task must fail closed.",
			},
		},
	); err == nil || !strings.Contains(err.Error(), "wave.step_id is required") {
		t.Fatalf("team task without Wave step did not fail closed: %v", err)
	}
	if _, err := handler.registry.Call(
		"dev.task.create",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":            "task-unknown-wave-step",
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"wave": map[string]interface{}{
				"step_id": "PILOT-W00-S99",
			},
			"title": "Unknown Wave step",
			"request": map[string]interface{}{
				"raw_request": "This undeclared step must fail closed.",
			},
		},
	); err == nil || !strings.Contains(err.Error(), "is not declared") {
		t.Fatalf("team task accepted an undeclared Wave step: %v", err)
	}

	result, err := handler.registry.Call(
		"dev.task.create",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":            "task-server-authority",
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"base_ref":      "main",
			"wave": map[string]interface{}{
				"step_id":         "PILOT-W00-S03",
				"wave_id":         "CLIENT-FORGED",
				"plan_revision":   99,
				"plan_path":       "docs/waves/forged.md",
				"registry_sha256": strings.Repeat("0", 64),
				"plan_sha256":     strings.Repeat("f", 64),
			},
			"title":       "Server authority",
			"repo_path":   "/tmp/client-controlled",
			"created_by":  fixture.alice.ID,
			"assignee_id": fixture.bob.ID,
			"request": map[string]interface{}{
				"raw_request": "implement the assigned change",
			},
			"plan": map[string]interface{}{
				"milestones": []map[string]interface{}{{
					"id":    "server-authority",
					"title": "Server authority",
					"work_items": []map[string]interface{}{{
						"id":           workItem.ID,
						"title":        workItem.Title,
						"instructions": workItem.Instructions,
					}},
				}},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	task := result.(dev.Task)
	if task.CreatedBy != teamcontrol.PlannerServicePrincipal ||
		task.RequestedBy != fixture.bob.ID ||
		task.AssigneeID != fixture.bob.ID {
		t.Fatalf("planner, requester, or validated assignee was not bound correctly: %+v", task)
	}
	if task.RepoPath != filepath.Clean(fixture.repoDir) {
		t.Fatalf("client repo path was trusted: %q", task.RepoPath)
	}
	if task.TeamID != fixture.team.ID {
		t.Fatalf("team was not resolved server-side: %q", task.TeamID)
	}
	expectedWave := gatewayPilotWaveBinding(t, fixture.repoDir)
	if task.Wave == nil || *task.Wave != *expectedWave {
		t.Fatalf("client Wave authority was not replaced server-side: %+v", task.Wave)
	}
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = fixture.repoDir
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	if task.Compile.BaseRef != strings.TrimSpace(string(output)) {
		t.Fatalf(
			"task base_ref was not pinned to the resolved commit: %q",
			task.Compile.BaseRef,
		)
	}
	revisedResult, err := handler.registry.Call(
		"dev.task.revise",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":                task.ID,
			"reason":            "Clarify the implementation after review feedback.",
			"expected_revision": task.Compile.Revision,
			"replacement": map[string]interface{}{
				"title":         "Revised server authority",
				"project_id":    fixture.other.ID,
				"repository_id": "client-repository",
				"assignee_id":   fixture.viewer.ID,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	revised := revisedResult.(dev.Task)
	if revised.Title != "Revised server authority" ||
		revised.ProjectID != fixture.project.ID ||
		revised.RepositoryID != fixture.repo.ID ||
		revised.AssigneeID != fixture.bob.ID ||
		revised.Compile.Revision <= task.Compile.Revision {
		t.Fatalf("revised task escaped server-owned identity: %+v", revised)
	}
	if _, err := handler.registry.Call(
		"dev.task.create",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":            "task-invalid-assignee",
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"assignee_id":   fixture.viewer.ID,
			"title":         "Invalid assignee",
			"request": map[string]interface{}{
				"raw_request": "must not be assigned cross-project",
			},
		},
	); err == nil {
		t.Fatal("development task accepted a non-member assignee")
	}
	if _, err := handler.registry.Call(
		"dev.task.create",
		teamSessionID(fixture.viewer.ID),
		map[string]interface{}{
			"id":            "task-viewer-denied",
			"project_id":    fixture.project.ID,
			"repository_id": fixture.repo.ID,
			"title":         "Denied",
			"request": map[string]interface{}{
				"raw_request": "must not be created",
			},
		},
	); err == nil {
		t.Fatal("cross-project viewer created a development task")
	}
	if _, err := fixture.service.AddProjectMember(
		fixture.alice.ID,
		fixture.project.ID,
		teamcontrol.AddProjectMemberInput{
			UserID: fixture.viewer.ID,
			Role:   teamcontrol.ProjectDeveloper,
		},
	); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{
		"dev.task.run",
		"dev.task.repair",
		"dev.task.resume",
	} {
		if _, err := handler.registry.Call(
			method,
			teamSessionID(fixture.bob.ID),
			map[string]interface{}{"id": revised.ID, "force": true},
		); err == nil ||
			!strings.Contains(err.Error(), "dev.task.enqueue") ||
			!strings.Contains(err.Error(), "persistent runner queue") {
			t.Fatalf("%s did not fail closed in team mode: %v", method, err)
		}
	}
}

func TestTeamDevelopmentCancellationClosesLinkedResourcesIdempotently(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	issue, err := fixture.service.CreateIssue(
		fixture.bob.ID,
		teamcontrol.CreateIssueInput{
			ID:        "issue-cancel",
			ProjectID: fixture.project.ID,
			Type:      teamcontrol.IssueTask,
			Title:     "Cancel linked execution",
			Severity:  teamcontrol.SeverityMedium,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []teamcontrol.IssueStatus{
		teamcontrol.IssueTriaged,
		teamcontrol.IssueInProgress,
		teamcontrol.IssueVerifying,
	} {
		issue, err = fixture.service.TransitionIssue(
			fixture.bob.ID,
			fixture.project.ID,
			issue.ID,
			status,
			"",
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	workItem, err := fixture.service.CreateWorkItem(
		fixture.bob.ID,
		teamcontrol.CreateWorkItemInput{
			ID:           "work-cancel",
			ProjectID:    fixture.project.ID,
			IssueID:      issue.ID,
			Title:        "Cancel work",
			Instructions: "Exercise cancellation propagation.",
			Priority:     teamcontrol.PriorityP2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []teamcontrol.WorkItemStatus{
		teamcontrol.WorkItemInProgress,
		teamcontrol.WorkItemVerifying,
	} {
		workItem, err = fixture.service.TransitionWorkItem(
			fixture.bob.ID,
			fixture.project.ID,
			workItem.ID,
			status,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	development, err := dev.NewService(dev.Config{
		Root:     t.TempDir(),
		RepoPath: fixture.repoDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := development.CreateTask(dev.CreateRequest{
		ID:           "task-cancel",
		TeamID:       fixture.team.ID,
		ProjectID:    fixture.project.ID,
		RepositoryID: fixture.repo.ID,
		AssigneeID:   fixture.bob.ID,
		IssueIDs:     []string{issue.ID},
		Title:        "Cancel linked execution",
		RepoPath:     fixture.repoDir,
		BaseRef:      "main",
		Request:      dev.RequestFrame{RawRequest: "Cancel this task."},
		Plan: dev.PlanSpec{Milestones: []dev.Milestone{{
			ID: "cancel", Title: "Cancel", WorkItems: []dev.WorkItem{{
				ID:           workItem.ID,
				Title:        workItem.Title,
				Instructions: workItem.Instructions,
			}},
		}}},
		CreatedBy: fixture.bob.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetDevelopmentService(development)
	params := map[string]interface{}{
		"id":        task.ID,
		"reason":    "No longer in scope.",
		"rationale": "The project owner removed this work from the approved scope.",
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := handler.registry.Call(
			"dev.task.cancel",
			teamSessionID(fixture.alice.ID),
			params,
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.(dev.Task).Status != dev.TaskCancelled {
			t.Fatalf("cancelled task status = %q", result.(dev.Task).Status)
		}
	}
	workItem, err = fixture.service.GetWorkItem(
		fixture.bob.ID,
		fixture.project.ID,
		workItem.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if workItem.Status != teamcontrol.WorkItemCancelled {
		t.Fatalf("work item status = %q", workItem.Status)
	}
	issue, err = fixture.service.GetIssue(
		fixture.bob.ID,
		fixture.project.ID,
		issue.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Status != teamcontrol.IssueBlocked {
		t.Fatalf("issue status = %q", issue.Status)
	}
}

func TestTeamRepairRevisionRequiresQueueCancellationAndReentersExecution(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	issue, err := fixture.service.CreateIssue(
		fixture.bob.ID,
		teamcontrol.CreateIssueInput{
			ID:        "issue-revision-loop",
			ProjectID: fixture.project.ID,
			Type:      teamcontrol.IssueImprovement,
			Title:     "Repair revision loop",
			Severity:  teamcontrol.SeverityMedium,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	workItem, err := fixture.service.CreateWorkItem(
		fixture.bob.ID,
		teamcontrol.CreateWorkItemInput{
			ID:           "work-revision-loop",
			ProjectID:    fixture.project.ID,
			IssueID:      issue.ID,
			Title:        "Repair revision loop",
			Instructions: "Verify a cancelled queue can advance to a new revision.",
			Priority:     teamcontrol.PriorityP1,
			VerificationCommands: [][]string{
				{"git", "diff", "--check"},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Assign(
		fixture.bob.ID,
		teamcontrol.AssignInput{
			ID:         "assign-revision-loop",
			ProjectID:  fixture.project.ID,
			TargetType: teamcontrol.AssignmentWorkItem,
			TargetID:   workItem.ID,
			UserID:     fixture.bob.ID,
			Role:       teamcontrol.AssignmentOwner,
		},
	); err != nil {
		t.Fatal(err)
	}
	development, err := dev.NewService(dev.Config{
		Root:     t.TempDir(),
		RepoPath: fixture.repoDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := development.CreateTask(dev.CreateRequest{
		ID:                 "task-revision-loop",
		TeamID:             fixture.team.ID,
		ProjectID:          fixture.project.ID,
		RepositoryID:       fixture.repo.ID,
		AssigneeID:         fixture.bob.ID,
		IssueIDs:           []string{issue.ID},
		PolicyBundleHash:   fixture.policy.Hash,
		PolicyInstructions: []string{"Keep the repair scoped."},
		Wave:               gatewayPilotWaveBinding(t, fixture.repoDir),
		Title:              "Repair revision loop",
		RepoPath:           fixture.repoDir,
		BaseRef:            "main",
		Request: dev.RequestFrame{
			RawRequest: "Exercise a repair revision after queue cancellation.",
		},
		Plan: dev.PlanSpec{Milestones: []dev.Milestone{{
			ID: "repair", Title: "Repair", WorkItems: []dev.WorkItem{{
				ID:           workItem.ID,
				Title:        workItem.Title,
				Instructions: workItem.Instructions,
				VerificationCommands: []dev.CommandSpec{{
					Name: "git diff check",
					Argv: []string{"git", "diff", "--check"},
				}},
			}},
		}}},
		EvidencePlan: dev.EvidencePlan{Commands: []dev.CommandSpec{{
			Name: "git diff check",
			Argv: []string{"git", "diff", "--check"},
		}}},
		Scope: dev.ScopePolicy{
			AllowedPaths:    []string{"README.md"},
			MaxChangedFiles: 1,
			MaxChangedLines: 20,
		},
		CreatedBy: fixture.bob.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	approveAndFreeze := func(task dev.Task) dev.Task {
		t.Helper()
		for _, kind := range dev.RequiredReviewKinds {
			task, err = development.ReviewTask(
				task.ID,
				kind,
				dev.ReviewApproved,
				fixture.alice.ID,
				"approved for deterministic revision-loop verification",
			)
			if err != nil {
				t.Fatal(err)
			}
		}
		task, err = development.FreezeTask(
			context.Background(),
			task.ID,
			fixture.alice.ID,
		)
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	task = approveAndFreeze(task)
	queueRoot := t.TempDir()
	queue, err := workstation.NewService(workstation.Config{Root: queueRoot})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetWorkstationService(queue)
	handler.SetDevelopmentService(development)

	firstResult, err := handler.registry.Call(
		"dev.task.enqueue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":      task.ID,
			"capabilities": []string{"codex"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	first := firstResult.(workstation.Task)
	if first.ID != "task-revision-loop-r1" {
		t.Fatalf("first queue id = %q", first.ID)
	}
	if _, err := handler.registry.Call(
		"dev.task.revise",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":                task.ID,
			"expected_revision": 1,
			"reason":            "Must not revise while the old queue is active.",
		},
	); err == nil || !strings.Contains(err.Error(), "still queued") {
		t.Fatalf("active queue revision error = %v", err)
	}
	if _, err := handler.registry.Call(
		"runner.cancel",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":         first.ID,
			"idempotency_key": "cancel-revision-one",
			"reason":          "Create the reviewed repair revision.",
		},
	); err != nil {
		t.Fatal(err)
	}
	revisedResult, err := handler.registry.Call(
		"dev.task.revise",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":                task.ID,
			"expected_revision": 1,
			"reason":            "Create a new reviewed repair revision.",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	revised := revisedResult.(dev.Task)
	if revised.Compile.Revision != 2 ||
		revised.Status != dev.TaskReviewPending {
		t.Fatalf("revised task = %+v", revised)
	}
	blocked, err := fixture.service.GetWorkItem(
		fixture.bob.ID,
		fixture.project.ID,
		workItem.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Status != teamcontrol.WorkItemBlocked {
		t.Fatalf("work item after revision = %q", blocked.Status)
	}
	revised = approveAndFreeze(revised)
	secondResult, err := handler.registry.Call(
		"dev.task.enqueue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":      revised.ID,
			"capabilities": []string{"codex"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	second := secondResult.(workstation.Task)
	if second.ID != "task-revision-loop-r2" ||
		second.Status != workstation.TaskQueued {
		t.Fatalf("second queue task = %+v", second)
	}
	inProgress, err := fixture.service.GetWorkItem(
		fixture.bob.ID,
		fixture.project.ID,
		workItem.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if inProgress.Status != teamcontrol.WorkItemInProgress {
		t.Fatalf("work item did not re-enter execution: %q", inProgress.Status)
	}
}

func TestSharedIssueResolvesOnlyAfterAllDevelopmentWorkIsDone(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	issue, err := fixture.service.CreateIssue(
		fixture.bob.ID,
		teamcontrol.CreateIssueInput{
			ID:        "issue-shared-gate",
			ProjectID: fixture.project.ID,
			Type:      teamcontrol.IssueBug,
			Title:     "Shared implementation gate",
			Severity:  teamcontrol.SeverityHigh,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []teamcontrol.IssueStatus{
		teamcontrol.IssueTriaged,
		teamcontrol.IssueInProgress,
		teamcontrol.IssueVerifying,
	} {
		issue, err = fixture.service.TransitionIssue(
			fixture.bob.ID,
			fixture.project.ID,
			issue.ID,
			status,
			"",
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	workItems := make([]teamcontrol.WorkItem, 0, 2)
	for index := 1; index <= 2; index++ {
		item, err := fixture.service.CreateWorkItem(
			fixture.bob.ID,
			teamcontrol.CreateWorkItemInput{
				ID:           fmt.Sprintf("work-shared-%d", index),
				ProjectID:    fixture.project.ID,
				IssueID:      issue.ID,
				Title:        fmt.Sprintf("Shared work %d", index),
				Instructions: "Implement one independent part.",
				Priority:     teamcontrol.PriorityP1,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, status := range []teamcontrol.WorkItemStatus{
			teamcontrol.WorkItemInProgress,
			teamcontrol.WorkItemVerifying,
		} {
			item, err = fixture.service.TransitionWorkItem(
				fixture.bob.ID,
				fixture.project.ID,
				item.ID,
				status,
			)
			if err != nil {
				t.Fatal(err)
			}
		}
		workItems = append(workItems, item)
	}
	development, err := dev.NewService(dev.Config{
		Root:     t.TempDir(),
		RepoPath: fixture.repoDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	tasks := make([]dev.Task, 0, 2)
	for index, item := range workItems {
		task, err := development.CreateTask(dev.CreateRequest{
			ID:           fmt.Sprintf("task-shared-%d", index+1),
			ProjectID:    fixture.project.ID,
			RepositoryID: fixture.repo.ID,
			AssigneeID:   fixture.bob.ID,
			IssueIDs:     []string{issue.ID},
			Title:        item.Title,
			RepoPath:     fixture.repoDir,
			Request:      dev.RequestFrame{RawRequest: item.Instructions},
			Plan: dev.PlanSpec{Milestones: []dev.Milestone{{
				ID: "shared", Title: "Shared", WorkItems: []dev.WorkItem{{
					ID:           item.ID,
					Title:        item.Title,
					Instructions: item.Instructions,
				}},
			}}},
			CreatedBy: fixture.bob.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		tasks = append(tasks, task)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetDevelopmentService(development)
	// Model the Done task value passed by the acceptance handler while keeping
	// the second task live in the service so the shared Issue remains gated.
	tasks[0].Status = dev.TaskDone
	if err := handler.transitionAcceptedDevelopmentResources(
		fixture.alice.ID,
		tasks[0],
	); err != nil {
		t.Fatal(err)
	}
	issue, err = fixture.service.GetIssue(
		fixture.alice.ID,
		fixture.project.ID,
		issue.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Status != teamcontrol.IssueVerifying {
		t.Fatalf("shared issue resolved before all work was done: %q", issue.Status)
	}
	if _, err := development.CancelTask(
		tasks[1].ID,
		fixture.alice.ID,
		"Second implementation was removed from scope.",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.TransitionWorkItem(
		fixture.alice.ID,
		fixture.project.ID,
		workItems[1].ID,
		teamcontrol.WorkItemBlocked,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.TransitionWorkItem(
		fixture.alice.ID,
		fixture.project.ID,
		workItems[1].ID,
		teamcontrol.WorkItemCancelled,
	); err != nil {
		t.Fatal(err)
	}
	if err := handler.transitionAcceptedDevelopmentResources(
		fixture.alice.ID,
		tasks[0],
	); err != nil {
		t.Fatal(err)
	}
	issue, err = fixture.service.GetIssue(
		fixture.alice.ID,
		fixture.project.ID,
		issue.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Status != teamcontrol.IssueVerifying {
		t.Fatalf("shared issue resolved after linked work was cancelled: %q", issue.Status)
	}
}

func TestRunnerRoleBoundaryAndFrozenBridgeRejectsTampering(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	issue, err := fixture.service.CreateIssue(
		fixture.bob.ID,
		teamcontrol.CreateIssueInput{
			ID:        "issue-alpha",
			ProjectID: fixture.project.ID,
			Type:      teamcontrol.IssueBug,
			Title:     "Alpha is broken",
			Severity:  teamcontrol.SeverityHigh,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []teamcontrol.IssueStatus{
		teamcontrol.IssueTriaged,
		teamcontrol.IssueInProgress,
	} {
		issue, err = fixture.service.TransitionIssue(
			fixture.bob.ID,
			fixture.project.ID,
			issue.ID,
			status,
			"",
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	workItem, err := fixture.service.CreateWorkItem(
		fixture.bob.ID,
		teamcontrol.CreateWorkItemInput{
			ID:           "work-alpha",
			ProjectID:    fixture.project.ID,
			IssueID:      issue.ID,
			Title:        "Implement alpha",
			Instructions: "Change alpha",
			Priority:     teamcontrol.PriorityP1,
			VerificationCommands: [][]string{
				{"go", "test", "./..."},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Assign(
		fixture.bob.ID,
		teamcontrol.AssignInput{
			ID:         "assign-work-alpha-owner",
			ProjectID:  fixture.project.ID,
			TargetType: teamcontrol.AssignmentWorkItem,
			TargetID:   workItem.ID,
			UserID:     fixture.bob.ID,
			Role:       teamcontrol.AssignmentOwner,
		},
	); err != nil {
		t.Fatal(err)
	}
	development, err := dev.NewService(dev.Config{
		Root:                      t.TempDir(),
		RepoPath:                  fixture.repoDir,
		RequireHumanFinalApproval: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := development.CreateTask(dev.CreateRequest{
		ID:                 "task-alpha",
		TeamID:             fixture.team.ID,
		ProjectID:          fixture.project.ID,
		RepositoryID:       fixture.repo.ID,
		AssigneeID:         fixture.bob.ID,
		IssueIDs:           []string{issue.ID},
		PolicyBundleHash:   fixture.policy.Hash,
		PolicyInstructions: []string{"Run gofmt."},
		Wave:               gatewayPilotWaveBinding(t, fixture.repoDir),
		Title:              "Implement alpha",
		RepoPath:           fixture.repoDir,
		BaseRef:            "main",
		Request:            dev.RequestFrame{RawRequest: "Implement alpha"},
		Plan: dev.PlanSpec{Milestones: []dev.Milestone{{
			ID: "m1", Title: "M1", WorkItems: []dev.WorkItem{{
				ID:           workItem.ID,
				Title:        workItem.Title,
				Instructions: workItem.Instructions,
				VerificationCommands: []dev.CommandSpec{{
					Name: "go test", Argv: []string{"go", "test", "./..."},
				}},
			}},
		}}},
		EvidencePlan: dev.EvidencePlan{Commands: []dev.CommandSpec{{
			Name: "go test", Argv: []string{"go", "test", "./..."},
		}}},
		Scope: dev.ScopePolicy{
			AllowedPaths:    []string{"README.md"},
			MaxChangedFiles: 1,
			MaxChangedLines: 20,
		},
		CreatedBy: fixture.bob.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range dev.RequiredReviewKinds {
		task, err = development.ReviewTask(
			task.ID,
			kind,
			dev.ReviewApproved,
			fixture.alice.ID,
			"approved with deterministic evidence",
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	task, err = development.FreezeTask(
		context.Background(),
		task.ID,
		fixture.alice.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	queueRoot := t.TempDir()
	queue, err := workstation.NewService(workstation.Config{Root: queueRoot})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetWorkstationService(queue)
	handler.SetDevelopmentService(development)

	key, err := workstation.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	registerParams := map[string]interface{}{
		"id":           "runner-bob",
		"name":         "Bob PC",
		"projects":     []string{fixture.project.ID},
		"capabilities": []string{"codex", "goclaw-runtime-linux-v1"},
		"metadata": map[string]string{
			"runner_goos":       "linux",
			"runner_goarch":     "amd64",
			"host_profile":      "native-linux",
			"isolation_backend": "bwrap",
			"sandbox_sha256":    strings.Repeat("a", 64),
		},
		"device_key": base64.RawURLEncoding.EncodeToString(key),
	}
	if _, err := handler.registry.Call(
		"runner.register",
		teamSessionID(fixture.bob.ID),
		registerParams,
	); err != nil {
		t.Fatalf("developer could not register own runner: %v", err)
	}
	listedResult, err := handler.registry.Call(
		"runner.list",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{"project_id": fixture.project.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	listed := listedResult.([]map[string]interface{})
	if len(listed) != 1 ||
		listed[0]["member_id"] != fixture.bob.ID {
		t.Fatalf("runner projection lost its owner: %+v", listed)
	}
	projectedMetadata, ok := listed[0]["metadata"].(map[string]string)
	if !ok ||
		projectedMetadata["sandbox_sha256"] != strings.Repeat("a", 64) {
		t.Fatalf("runner projection lost platform metadata: %+v", listed[0])
	}
	registerParams["id"] = "runner-viewer"
	registerParams["projects"] = []string{fixture.other.ID}
	if _, err := handler.registry.Call(
		"runner.register",
		teamSessionID(fixture.viewer.ID),
		registerParams,
	); err == nil {
		t.Fatal("viewer registered a development runner")
	}
	if _, err := handler.registry.Call(
		"runner.update",
		teamSessionID(fixture.viewer.ID),
		map[string]interface{}{
			"runner_id": "runner-bob",
			"name":      "Hijacked",
		},
	); err == nil {
		t.Fatal("non-owner updated another member's runner")
	}
	if _, err := handler.registry.Call(
		"runner.update",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"runner_id": "runner-bob",
			"projects":  []string{fixture.other.ID},
		},
	); err == nil {
		t.Fatal("runner owner added a project without membership")
	}
	updatedRunner, err := handler.registry.Call(
		"runner.update",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"runner_id": "runner-bob",
			"name":      "Bob Secure PC",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updatedRunner.(workstation.Runner).Name != "Bob Secure PC" {
		t.Fatalf("runner name was not updated: %+v", updatedRunner)
	}

	rawPack := workstation.ExecutionPack{
		ProjectID:    fixture.project.ID,
		RepositoryID: fixture.repo.ID,
		BaseCommit:   task.Compile.BaseCommit,
		Prompt:       "client controlled",
		Verification: []workstation.CommandSpec{{Name: "false", Argv: []string{"false"}}},
	}
	if _, err := handler.registry.Call(
		"runner.enqueue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"idempotency_key": "raw-bypass",
			"project_id":      fixture.project.ID,
			"execution_pack":  rawPack,
		},
	); err == nil {
		t.Fatal("developer bypassed frozen task with raw runner.enqueue")
	}
	if _, err := handler.registry.Call(
		"runner.enqueue",
		teamSessionID(fixture.alice.ID),
		map[string]interface{}{
			"idempotency_key": "raw-manager-bypass",
			"project_id":      fixture.project.ID,
			"execution_pack":  rawPack,
		},
	); err == nil || !strings.Contains(err.Error(), "raw runner.enqueue is disabled") {
		t.Fatalf("project manager bypassed the frozen task factory: %v", err)
	}

	result, err := handler.registry.Call(
		"dev.task.enqueue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":      task.ID,
			"capabilities": []string{"codex"},
			"execution_pack": map[string]interface{}{
				"prompt":        "malicious client prompt",
				"allowed_paths": []string{"/"},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	queued := result.(workstation.Task)
	if strings.Contains(queued.ExecutionPack.Prompt, "malicious") {
		t.Fatal("dev.task.enqueue trusted a client execution pack")
	}
	if queued.ExecutionPack.RepositoryURL != fixture.repo.RemoteURL ||
		queued.ExecutionPack.PolicyBundleHash != fixture.policy.Hash {
		t.Fatalf("server authority was not applied to execution pack: %+v", queued.ExecutionPack)
	}
	if !containsDevelopmentID(
		queued.RequiredCapabilities,
		"goclaw-runtime-linux-v1",
	) {
		t.Fatalf("queue does not require the Linux runtime contract: %+v", queued.RequiredCapabilities)
	}
	if queued.ExecutionPack.ExecutionProfile != workstation.ExecutionProfileStrict {
		t.Fatalf(
			"default execution profile = %q",
			queued.ExecutionPack.ExecutionProfile,
		)
	}
	if queued.ExecutionPack.Metadata["wave_id"] != task.Wave.WaveID ||
		queued.ExecutionPack.Metadata["wave_step"] != task.Wave.StepID ||
		queued.ExecutionPack.Metadata["wave_plan_sha256"] != task.Wave.PlanSHA256 {
		t.Fatalf("execution pack lost the immutable Wave binding: %+v", queued.ExecutionPack.Metadata)
	}
	queuedProjectionPath := filepath.Join(queueRoot, "tasks", queued.ID+".json")
	queuedProjection, err := os.ReadFile(queuedProjectionPath)
	if err != nil {
		t.Fatal(err)
	}
	var tamperedQueued workstation.Task
	if err := json.Unmarshal(queuedProjection, &tamperedQueued); err != nil {
		t.Fatal(err)
	}
	delete(tamperedQueued.ExecutionPack.Metadata, "wave_step")
	tamperedQueuedProjection, err := json.MarshalIndent(
		tamperedQueued,
		"",
		"  ",
	)
	if err != nil {
		t.Fatal(err)
	}
	tamperedQueuedProjection = append(tamperedQueuedProjection, '\n')
	if err := os.WriteFile(
		queuedProjectionPath,
		tamperedQueuedProjection,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"dev.task.enqueue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":      task.ID,
			"capabilities": []string{"codex"},
		},
	); err == nil || !strings.Contains(err.Error(), "wave_step") {
		t.Fatalf("enqueue replay trusted incomplete Wave metadata: %v", err)
	}
	if err := os.WriteFile(
		queuedProjectionPath,
		queuedProjection,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	replayed, err := handler.registry.Call(
		"dev.task.enqueue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"id":              "alternate-client-id",
			"task_id":         task.ID,
			"idempotency_key": "alternate-client-key",
			"capabilities":    []string{"codex"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.(workstation.Task).ID != queued.ID {
		t.Fatal("alternate client identity duplicated a frozen task revision")
	}
	updated, err := fixture.service.GetWorkItem(
		fixture.bob.ID,
		fixture.project.ID,
		workItem.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != teamcontrol.WorkItemInProgress {
		t.Fatalf("enqueue did not advance work item: %s", updated.Status)
	}

	claimResult, err := handler.registry.Call(
		"runner.claim",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"runner_id":       "runner-bob",
			"project_id":      fixture.project.ID,
			"idempotency_key": "claim-task-alpha",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := claimResult.(workstation.ClaimResult)
	now := time.Now().UTC()
	diffPatch := strings.Join([]string{
		"diff --git a/README.md b/README.md",
		"--- a/README.md",
		"+++ b/README.md",
		"@@ -1 +1 @@",
		"-# test",
		"+# changed by workstation",
		"",
	}, "\n")
	diffSHA256 := fmt.Sprintf("%x", sha256.Sum256([]byte(diffPatch)))
	evidence, err := workstation.SignEvidenceBundle(
		workstation.EvidenceBundle{
			TaskID:              claim.Task.ID,
			ProjectID:           claim.Task.ProjectID,
			ExecutionPackSHA256: claim.Task.ExecutionPackSHA256,
			RunnerID:            "runner-bob",
			LeaseID:             claim.Lease.ID,
			Attempt:             claim.Lease.Attempt,
			Outcome:             "completed",
			StartedAt:           now.Add(-time.Second),
			FinishedAt:          now,
			BaseCommit:          claim.Task.ExecutionPack.BaseCommit,
			HeadCommit:          claim.Task.ExecutionPack.BaseCommit,
			ChangedFiles:        []string{"README.md"},
			DiffPatch:           diffPatch,
			DiffSHA256:          diffSHA256,
			Checks: []workstation.EvidenceCheck{
				{Name: "runner-setup", Passed: true},
				{Name: "codex-exec", Passed: true},
				{Name: "go test", Passed: true},
				{Name: "scope-policy", Passed: true},
				{Name: "no-automatic-commit", Passed: true},
			},
		},
		key,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"runner.complete",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"runner_id":       "runner-bob",
			"task_id":         claim.Task.ID,
			"lease_id":        claim.Lease.ID,
			"idempotency_key": "complete-task-alpha",
			"summary":         "completed with signed evidence",
			"evidence":        evidence,
		},
	); err != nil {
		t.Fatal(err)
	}
	updated, err = fixture.service.GetWorkItem(
		fixture.bob.ID,
		fixture.project.ID,
		workItem.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != teamcontrol.WorkItemVerifying {
		t.Fatalf("completion did not move work item to verifying: %s", updated.Status)
	}
	imported, err := development.GetTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Status != dev.TaskAwaitingAcceptance ||
		imported.LastGate == nil ||
		!imported.LastGate.Passed {
		t.Fatalf("workstation evidence did not pass development DoneGate: %+v", imported)
	}
	evidenceTask, evidencePack, importedDiff, err := handler.readTaskEvidence(task.ID)
	if err != nil {
		t.Fatalf("read imported development evidence: %v", err)
	}
	if evidenceTask.ID != task.ID ||
		evidencePack.Imported == nil ||
		importedDiff != diffPatch {
		t.Fatalf(
			"imported evidence projection mismatch: task=%s pack=%+v diff=%q",
			evidenceTask.ID,
			evidencePack.Imported,
			importedDiff,
		)
	}
	issue, err = fixture.service.GetIssue(
		fixture.bob.ID,
		fixture.project.ID,
		issue.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Status != teamcontrol.IssueVerifying {
		t.Fatalf("completion did not move issue to verifying: %s", issue.Status)
	}
	queueProjectionPath := filepath.Join(
		queueRoot,
		"tasks",
		claim.Task.ID+".json",
	)
	queueProjection, err := os.ReadFile(queueProjectionPath)
	if err != nil {
		t.Fatal(err)
	}
	var tamperedQueue workstation.Task
	if err := json.Unmarshal(queueProjection, &tamperedQueue); err != nil {
		t.Fatal(err)
	}
	delete(tamperedQueue.ExecutionPack.Metadata, "wave_step")
	tamperedProjection, err := json.MarshalIndent(tamperedQueue, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tamperedProjection = append(tamperedProjection, '\n')
	if err := os.WriteFile(
		queueProjectionPath,
		tamperedProjection,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"dev.task.accept",
		teamSessionID(fixture.alice.ID),
		map[string]interface{}{
			"id":        task.ID,
			"rationale": "Tampered Wave metadata must fail closed.",
		},
	); err == nil || !strings.Contains(err.Error(), "wave_step") {
		t.Fatalf("accept trusted an incomplete execution pack Wave binding: %v", err)
	}
	if err := os.WriteFile(
		queueProjectionPath,
		queueProjection,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	var acceptedResult interface{}
	for attempt := 0; attempt < 2; attempt++ {
		acceptedResult, err = handler.registry.Call(
			"dev.task.accept",
			teamSessionID(fixture.alice.ID),
			map[string]interface{}{
				"id":        task.ID,
				"rationale": "Signed evidence and DoneGate results satisfy the task.",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	if acceptedResult.(dev.Task).Status != dev.TaskDone {
		t.Fatalf("accepted task did not close: %+v", acceptedResult)
	}
	updated, err = fixture.service.GetWorkItem(
		fixture.bob.ID,
		fixture.project.ID,
		workItem.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != teamcontrol.WorkItemDone {
		t.Fatalf("acceptance did not close work item: %s", updated.Status)
	}
	issue, err = fixture.service.GetIssue(
		fixture.bob.ID,
		fixture.project.ID,
		issue.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Status != teamcontrol.IssueResolved {
		t.Fatalf("acceptance did not resolve issue: %s", issue.Status)
	}
	acceptedTask := acceptedResult.(dev.Task)
	if err := os.WriteFile(
		filepath.Join(fixture.repoDir, "README.md"),
		[]byte("# changed by workstation\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	commitMessage := strings.Join([]string{
		"Apply accepted workstation patch",
		"",
		"Task-ID: " + acceptedTask.ID,
		"Project-ID: " + acceptedTask.ProjectID,
		fmt.Sprintf("Task-Revision: %d", acceptedTask.Compile.Revision),
		"Repository-ID: " + acceptedTask.RepositoryID,
		"Correlation-ID: " + acceptedTask.CorrelationID,
		"Policy-Bundle: " + acceptedTask.PolicyBundleHash,
		"Wave-ID: " + acceptedTask.Wave.WaveID,
		fmt.Sprintf("Wave-Revision: %d", acceptedTask.Wave.PlanRevision),
		"Wave-Step: " + acceptedTask.Wave.StepID,
		"Work-Item: " + workItem.ID,
		"Issue: " + issue.ID,
	}, "\n")
	for _, args := range [][]string{
		{"add", "README.md"},
		{"commit", "-m", commitMessage},
	} {
		command := exec.Command("git", args...)
		command.Dir = fixture.repoDir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = fixture.repoDir
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	commitSHA := strings.TrimSpace(string(output))
	linkParams := map[string]interface{}{
		"id":         acceptedTask.ID,
		"commit_sha": commitSHA,
		"url":        "https://example.test/pull/42",
	}
	for attempt := 0; attempt < 2; attempt++ {
		linkedResult, err := handler.registry.Call(
			"dev.task.link-pr",
			teamSessionID(fixture.bob.ID),
			linkParams,
		)
		if err != nil {
			t.Fatal(err)
		}
		linked := linkedResult.(dev.Task)
		if linked.CommitSHA != commitSHA ||
			linked.PullRequestURL != "https://example.test/pull/42" {
			t.Fatalf("external commit/PR was not linked: %+v", linked)
		}
	}
	artifacts, err := fixture.service.ListArtifacts(
		fixture.bob.ID,
		fixture.project.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !artifactExists(artifacts, "commit-"+commitSHA) ||
		!artifactExists(artifacts, "pr-"+commitSHA) {
		t.Fatalf("external commit/PR artifacts were not registered: %+v", artifacts)
	}
	links, err := fixture.service.ListLinks(
		fixture.bob.ID,
		fixture.project.ID,
		teamcontrol.ResourceCommit,
		"commit-"+commitSHA,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) < 3 {
		t.Fatalf("external commit correlations are incomplete: %+v", links)
	}
	rotatedKey, err := workstation.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler.registry.Call(
		"runner.key.rotate",
		teamSessionID(fixture.viewer.ID),
		map[string]interface{}{
			"runner_id":  "runner-bob",
			"device_key": base64.RawURLEncoding.EncodeToString(rotatedKey),
		},
	); err == nil {
		t.Fatal("non-owner rotated another member's runner key")
	}
	rotatedResult, err := handler.registry.Call(
		"runner.key.rotate",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"runner_id":  "runner-bob",
			"device_key": base64.RawURLEncoding.EncodeToString(rotatedKey),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	rotated := rotatedResult.(workstation.Runner)
	oldKeyID, _ := workstation.DeviceKeyID(key)
	if rotated.KeyID == oldKeyID {
		t.Fatal("runner key rotation did not change the public key id")
	}
	for _, method := range []string{
		"team.members",
		"work.items",
		"issue.list",
		"runner.list",
		"policy.status",
		"docs.summary",
		"components.summary",
	} {
		if _, err := handler.registry.Call(
			method,
			teamSessionID(fixture.bob.ID),
			map[string]interface{}{"project_id": fixture.project.ID},
		); err != nil {
			t.Fatalf("%s projection failed: %v", method, err)
		}
	}
}

func TestRunnerRequeueRequiresAssigneeAndForceRequiresManager(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	if _, err := fixture.service.AddProjectMember(
		fixture.alice.ID,
		fixture.project.ID,
		teamcontrol.AddProjectMemberInput{
			UserID: fixture.viewer.ID,
			Role:   teamcontrol.ProjectDeveloper,
		},
	); err != nil {
		t.Fatal(err)
	}
	queue, err := workstation.NewService(workstation.Config{
		Root: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	key, err := workstation.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.RegisterRunner(
		workstation.RegisterRunnerRequest{
			ID:           "runner-requeue",
			Name:         "Requeue runner",
			OwnerUserID:  fixture.bob.ID,
			Capabilities: []string{"codex"},
			Projects:     []string{fixture.project.ID},
		},
		key,
	); err != nil {
		t.Fatal(err)
	}
	queued, err := queue.Enqueue(workstation.EnqueueRequest{
		ID:             "wstask-requeue-auth",
		IdempotencyKey: "enqueue-requeue-auth",
		ProjectID:      fixture.project.ID,
		MaxAttempts:    2,
		RequiredCapabilities: []string{
			"codex",
		},
		ExecutionPack: workstation.ExecutionPack{
			ProjectID:    fixture.project.ID,
			RepositoryID: fixture.repo.ID,
			BaseCommit:   strings.Repeat("a", 40),
			Prompt:       "fixture",
			Verification: []workstation.CommandSpec{{
				Name: "fixture", Argv: []string{"true"},
			}},
			Metadata: map[string]string{"assignee_id": fixture.bob.ID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	claim, err := queue.Claim(workstation.ClaimRequest{
		RunnerID:       "runner-requeue",
		ProjectID:      fixture.project.ID,
		IdempotencyKey: "claim-requeue-auth",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Fail(workstation.FailRequest{
		RunnerID:       "runner-requeue",
		TaskID:         queued.ID,
		LeaseID:        claim.Lease.ID,
		IdempotencyKey: "fail-requeue-auth",
		Error:          "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetWorkstationService(queue)
	if _, err := handler.registry.Call(
		"runner.requeue",
		teamSessionID(fixture.viewer.ID),
		map[string]interface{}{
			"task_id":         queued.ID,
			"idempotency_key": "viewer-requeue",
			"reason":          "not the assignee",
		},
	); err == nil {
		t.Fatal("non-assignee developer requeued another member's task")
	}
	cancelCandidate, err := queue.Enqueue(workstation.EnqueueRequest{
		ID:             "wstask-cancel-auth",
		IdempotencyKey: "enqueue-cancel-auth",
		ProjectID:      fixture.project.ID,
		ExecutionPack: workstation.ExecutionPack{
			ProjectID:    fixture.project.ID,
			RepositoryID: fixture.repo.ID,
			BaseCommit:   strings.Repeat("b", 40),
			Prompt:       "cancel fixture",
			Verification: []workstation.CommandSpec{{
				Name: "fixture-cancel", Argv: []string{"true"},
			}},
			Metadata: map[string]string{"assignee_id": fixture.bob.ID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	cancelParams := map[string]interface{}{
		"task_id":         cancelCandidate.ID,
		"idempotency_key": "cancel-auth",
		"reason":          "superseded before claim",
	}
	if _, err := handler.registry.Call(
		"runner.cancel",
		teamSessionID(fixture.viewer.ID),
		cancelParams,
	); err == nil {
		t.Fatal("non-assignee developer cancelled another member's queued task")
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := handler.registry.Call(
			"runner.cancel",
			teamSessionID(fixture.bob.ID),
			cancelParams,
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.(workstation.Task).Status != workstation.TaskCancelled {
			t.Fatalf("cancelled runner task = %+v", result)
		}
	}
	if _, err := handler.registry.Call(
		"runner.requeue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":         queued.ID,
			"idempotency_key": "assignee-force-requeue",
			"reason":          "force requires manager",
			"force":           true,
		},
	); err == nil {
		t.Fatal("assignee forced a requeue without project management permission")
	}
	result, err := handler.registry.Call(
		"runner.requeue",
		teamSessionID(fixture.bob.ID),
		map[string]interface{}{
			"task_id":         queued.ID,
			"idempotency_key": "assignee-requeue",
			"reason":          "retry the assigned task",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.(workstation.Task).Status != workstation.TaskQueued {
		t.Fatalf("assignee requeue result = %+v", result)
	}
}

func mustCreateUser(
	t *testing.T,
	service *teamcontrol.Service,
	id string,
) teamcontrol.User {
	t.Helper()
	user, err := service.CreateUser(teamcontrol.CreateUserInput{
		ID: id, DisplayName: strings.ToUpper(id[:1]) + id[1:],
	})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func initGatewayGitRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "gateway-test@example.com"},
		{"config", "user.name", "Gateway Test"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile(
		filepath.Join(root, "README.md"),
		[]byte("# test\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(
		filepath.Join(root, "docs", "waves", "pilot"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	plan := []byte("---\n" +
		"schema: goclaw.wave/v1\n" +
		"wave_id: PILOT-W00\n" +
		"revision: 1\n" +
		"plan_status: approved\n" +
		"wave_state: active\n" +
		"approved_by:\n  - gateway-test\n" +
		"depends_on:\n  - FE-W00\n" +
		"steps:\n  - PILOT-W00-S03\n" +
		"allowed_change_scope:\n  - README.md\n  - gateway/**\n" +
		"product_code_changes_allowed: true\n" +
		"---\n\n# Gateway pilot fixture\n")
	if err := os.WriteFile(
		filepath.Join(root, "docs", "waves", "pilot", "plan-r001.md"),
		plan,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	registry := []byte(`{
  "schema_version": 1,
  "active_wave": "PILOT-W00",
  "waves": [
    {
      "id": "FE-W00",
      "status": "complete",
      "document": "frontend-stability/fe-w00/plan-r001.md",
      "depends_on": [],
      "allowed_change_scope": ["docs/**"],
      "product_code_changes_allowed": false
    },
    {
      "id": "PILOT-W00",
      "status": "active",
      "document": "pilot/plan-r001.md",
      "depends_on": ["FE-W00"],
      "allowed_change_scope": ["README.md", "gateway/**"],
      "product_code_changes_allowed": true
    }
  ]
}
`)
	if err := os.WriteFile(
		filepath.Join(root, "docs", "waves", "wave-registry.json"),
		registry,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "add", ".")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	command = exec.Command("git", "commit", "-m", "initial")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}
	return root
}

func gatewayPilotWaveBinding(t *testing.T, repo string) *dev.WaveBinding {
	t.Helper()
	registry, err := os.ReadFile(
		filepath.Join(repo, "docs", "waves", "wave-registry.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := os.ReadFile(
		filepath.Join(repo, "docs", "waves", "pilot", "plan-r001.md"),
	)
	if err != nil {
		t.Fatal(err)
	}
	registrySHA := sha256.Sum256(registry)
	planSHA := sha256.Sum256(plan)
	return &dev.WaveBinding{
		WaveID:         "PILOT-W00",
		PlanRevision:   1,
		StepID:         "PILOT-W00-S03",
		PlanPath:       "docs/waves/pilot/plan-r001.md",
		RegistrySHA256: fmt.Sprintf("%x", registrySHA),
		PlanSHA256:     fmt.Sprintf("%x", planSHA),
	}
}

func newRPCRequestWithHeaders(userToken string) *http.Request {
	request := httptest.NewRequest("POST", "http://localhost/rpc", nil)
	if userToken != "" {
		request.Header.Set("X-GoClaw-User-Token", userToken)
	}
	return request
}
