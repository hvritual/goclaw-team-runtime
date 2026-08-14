package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeServesAuthorizedIssueReadSlice(t *testing.T) {
	now := time.Date(2026, 8, 13, 4, 5, 6, 0, time.UTC)
	sequence := 0
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "issue-read.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour, Now: func() time.Time { return now },
			NewID: func(context.Context) (string, error) {
				sequence++
				return []string{"user-one-token", "user-two-token"}[sequence-1], nil
			},
		},
	})
	userOne := verifyRuntimeUser(t, runtime, "one@example.com")
	userTwo := verifyRuntimeUser(t, runtime, "two@example.com")
	for _, statement := range []string{
		`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,created_at,updated_at) VALUES
			('workspace-one','One','one','{}','[]','ONE','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z'),
			('workspace-two','Two','two','{}','[]','TWO','2026-08-13T00:00:00Z','2026-08-13T00:00:00Z')`,
		`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES
			('member-one','workspace-one','` + userOne + `','member','2026-08-13T00:00:00Z'),
			('member-two','workspace-two','` + userTwo + `','member','2026-08-13T00:00:00Z')`,
		`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,description,status,priority,creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at) VALUES
			('issue-one','workspace-one',1,'ONE-1','First issue','Base detail','todo','high','member','` + userOne + `',1,'{}','{}','[]','2026-08-13T00:00:01Z','2026-08-13T00:00:01Z'),
			('issue-two','workspace-one',2,'ONE-2','Second issue',NULL,'done','low','member','` + userOne + `',2,'{}','{}','[]','2026-08-13T00:00:02Z','2026-08-13T00:00:02Z'),
			('issue-foreign','workspace-two',1,'TWO-1','Foreign issue',NULL,'todo','none','member','` + userTwo + `',1,'{}','{}','[]','2026-08-13T00:00:03Z','2026-08-13T00:00:03Z')`,
	} {
		if _, err := runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	headers := map[string]string{"Authorization": "Bearer user-one-token", "X-Workspace-Slug": "one"}

	missing := runtimeRequest(runtime, http.MethodGet, "/api/issues", "", nil)
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth = %d %s", missing.Code, missing.Body.String())
	}
	foreign := runtimeRequest(runtime, http.MethodGet, "/api/issues/issue-foreign", "", headers)
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign = %d %s", foreign.Code, foreign.Body.String())
	}

	list := runtimeRequest(runtime, http.MethodGet, "/api/issues?statuses=todo,done&limit=1&offset=1&sort=position&direction=asc", "", headers)
	if list.Code != http.StatusOK {
		t.Fatalf("list = %d %s", list.Code, list.Body.String())
	}
	var listBody struct {
		Issues []map[string]json.RawMessage `json:"issues"`
		Total  int                          `json:"total"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	if listBody.Total != 2 || len(listBody.Issues) != 1 || string(listBody.Issues[0]["identifier"]) != `"ONE-2"` {
		t.Fatalf("list body = %s", list.Body.String())
	}
	if _, camel := listBody.Issues[0]["workspaceId"]; camel {
		t.Fatalf("camelCase leaked: %s", list.Body.String())
	}
	for _, key := range []string{"workspace_id", "assignee_type", "creator_id", "parent_issue_id", "created_at", "metadata", "properties"} {
		if _, ok := listBody.Issues[0][key]; !ok {
			t.Fatalf("missing snake_case key %q: %s", key, list.Body.String())
		}
	}

	query := runtimeRequest(runtime, http.MethodPost, "/api/issues/query", `{"ids":"issue-two,issue-foreign"}`, headers)
	if query.Code != http.StatusOK || !json.Valid(query.Body.Bytes()) {
		t.Fatalf("query = %d %s", query.Code, query.Body.String())
	}
	var queryBody struct {
		Issues []struct {
			ID string `json:"id"`
		} `json:"issues"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(query.Body.Bytes(), &queryBody)
	if queryBody.Total != 1 || len(queryBody.Issues) != 1 || queryBody.Issues[0].ID != "issue-two" {
		t.Fatalf("query body = %s", query.Body.String())
	}

	detail := runtimeRequest(runtime, http.MethodGet, "/api/issues/ONE-1", "", headers)
	if detail.Code != http.StatusOK || !json.Valid(detail.Body.Bytes()) {
		t.Fatalf("detail = %d %s", detail.Code, detail.Body.String())
	}
	var detailBody map[string]json.RawMessage
	_ = json.Unmarshal(detail.Body.Bytes(), &detailBody)
	if string(detailBody["id"]) != `"issue-one"` || string(detailBody["description"]) != `"Base detail"` {
		t.Fatalf("detail body = %s", detail.Body.String())
	}

	facets := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/facets", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"facets":[{"kind":"status"},{"kind":"priority"}],"include_total":true}`, headers)
	if facets.Code != http.StatusOK || !containsJSON(facets.Body.Bytes(), `"key":"todo"`, `"key":"done"`, `"total":2`) {
		t.Fatalf("facets = %d %s", facets.Code, facets.Body.String())
	}
	groups := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/groups", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"status"}}`, headers)
	if groups.Code != http.StatusOK || !containsJSON(groups.Body.Bytes(), `"key":"status:todo"`, `"key":"status:done"`, `"total":2`) {
		t.Fatalf("groups = %d %s", groups.Code, groups.Body.String())
	}
	rows := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"status"},"group_key":"status:todo","hierarchy":{"enabled":false},"parent_id":null}`, headers)
	if rows.Code != http.StatusOK || !containsJSON(rows.Body.Bytes(), `"group_key":"status:todo"`, `"identifier":"ONE-1"`, `"branch_total":1`, `"total":0`) {
		t.Fatalf("rows = %d %s", rows.Code, rows.Body.String())
	}
	assigneeGroups := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/groups", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"assignee"}}`, headers)
	if assigneeGroups.Code != http.StatusOK || !containsJSON(assigneeGroups.Body.Bytes(), `"key":"assignee:unassigned"`) {
		t.Fatalf("assignee groups = %d %s", assigneeGroups.Code, assigneeGroups.Body.String())
	}

	if _, err := runtime.Database().Exec(`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,parent_issue_id,position,metadata,properties,asset_ids,created_at,updated_at) VALUES
		('issue-child','workspace-one',3,'ONE-3','Child issue','done','none','member','` + userOne + `','issue-one',3,'{}','{}','[]','2026-08-13T00:00:04Z','2026-08-13T00:00:04Z')`); err != nil {
		t.Fatal(err)
	}
	rootRows := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"status"},"group_key":"status:todo","hierarchy":{"enabled":true},"parent_id":null}`, headers)
	if rootRows.Code != http.StatusOK || !containsJSON(rootRows.Body.Bytes(), `"identifier":"ONE-1"`, `"direct_child_count":0`) {
		t.Fatalf("root rows = %d %s", rootRows.Code, rootRows.Body.String())
	}
	statusChildRows := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"status"},"group_key":"status:todo","hierarchy":{"enabled":true},"parent_id":"issue-one"}`, headers)
	if statusChildRows.Code != http.StatusOK || !containsJSON(statusChildRows.Body.Bytes(), `"branch_total":0`, `"rows":[]`) {
		t.Fatalf("status child rows = %d %s", statusChildRows.Code, statusChildRows.Body.String())
	}
	childRows := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":true},"parent_id":"issue-one"}`, headers)
	if childRows.Code != http.StatusOK || !containsJSON(childRows.Body.Bytes(), `"identifier":"ONE-3"`, `"branch_total":1`, `"total":0`) {
		t.Fatalf("child rows = %d %s", childRows.Code, childRows.Body.String())
	}
	actorFilter := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{"assignees":[{"type":"member","id":"`+userOne+`"}]},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null}`, headers)
	if actorFilter.Code != http.StatusOK {
		t.Fatalf("actor filter = %d %s", actorFilter.Code, actorFilter.Body.String())
	}
	unsupportedList := runtimeRequest(runtime, http.MethodGet, "/api/issues?label_ids=label-1", "", headers)
	if unsupportedList.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unsupported list filter = %d %s", unsupportedList.Code, unsupportedList.Body.String())
	}
	for _, shape := range []struct{ name, body string }{
		{"assignee-filter", `{"query":{"scope":{"kind":"workspace"},"filters":{"assignees":[{"type":"member","id":"` + userOne + `"}],"include_no_assignee":true},"sort":{"field":"status","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`},
		{"creator-filter", `{"query":{"scope":{"kind":"workspace"},"filters":{"creators":[{"type":"member","id":"` + userOne + `"}]},"sort":{"field":"priority","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`},
		{"project-date-filter", `{"query":{"scope":{"kind":"workspace"},"filters":{"project_ids":["project-one"],"include_no_project":true,"date":{"field":"created_at","start":"2026-08-12","end":"2026-08-14"}},"sort":{"field":"start_date","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`},
	} {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", shape.body, headers)
		if response.Code != http.StatusOK {
			t.Fatalf("controller shape %s = %d %s", shape.name, response.Code, response.Body.String())
		}
	}
	invalidDirection := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"title","direction":"sideways"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`, headers)
	if invalidDirection.Code != http.StatusBadRequest {
		t.Fatalf("invalid sort direction = %d %s", invalidDirection.Code, invalidDirection.Body.String())
	}
	for _, kind := range []string{"status", "assignee", "project", "parent"} {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/groups", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"due_date","direction":"asc"}},"group":{"kind":"`+kind+`"},"page":{"limit":50}}`, headers)
		if response.Code != http.StatusOK {
			t.Fatalf("group shape %s = %d %s", kind, response.Code, response.Body.String())
		}
	}
	for _, branch := range []struct{ kind, key string }{{"assignee", "assignee:unassigned"}, {"project", "project:none"}, {"parent", "parent:none"}} {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"`+branch.kind+`"},"group_key":"`+branch.key+`","hierarchy":{"enabled":false},"page":{"limit":50}}`, headers)
		if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"total":0`) {
			t.Fatalf("group row shape %s = %d %s", branch.kind, response.Code, response.Body.String())
		}
	}
	pagedGroups := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/groups", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"status"},"page":{"limit":1}}`, headers)
	if pagedGroups.Code != http.StatusOK || !containsJSON(pagedGroups.Body.Bytes(), `"next_cursor":"`) {
		t.Fatalf("paged groups = %d %s", pagedGroups.Code, pagedGroups.Body.String())
	}
	for _, cursor := range []string{"bogus", "-1"} {
		invalidCursor := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"page":{"cursor":"`+cursor+`"}}`, headers)
		if invalidCursor.Code != http.StatusBadRequest {
			t.Fatalf("cursor %q = %d %s", cursor, invalidCursor.Code, invalidCursor.Body.String())
		}
	}

	firstPage := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"page":{"limit":1}}`, headers)
	var pageBody struct {
		NextCursor *string `json:"next_cursor"`
	}
	if firstPage.Code != http.StatusOK || json.Unmarshal(firstPage.Body.Bytes(), &pageBody) != nil || pageBody.NextCursor == nil {
		t.Fatalf("first page = %d %s", firstPage.Code, firstPage.Body.String())
	}
	if !containsJSON(firstPage.Body.Bytes(), `"total":3`, `"branch_total":1`) {
		t.Fatalf("first page totals = %s", firstPage.Body.String())
	}
	secondPage := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"page":{"limit":1,"cursor":"`+*pageBody.NextCursor+`"}}`, headers)
	if secondPage.Code != http.StatusOK || !containsJSON(secondPage.Body.Bytes(), `"total":0`, `"branch_total":1`) {
		t.Fatalf("second page totals = %d %s", secondPage.Code, secondPage.Body.String())
	}
	badLowLimit := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"page":{"limit":0}}`, headers)
	if badLowLimit.Code != http.StatusBadRequest {
		t.Fatalf("low limit = %d %s", badLowLimit.Code, badLowLimit.Body.String())
	}
	for name, body := range map[string]string{
		"query":  `{"query":{"scope":{"kind":"workspace"},"filters":{"statuses":["todo"]},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"page":{"cursor":"` + *pageBody.NextCursor + `"}}`,
		"group":  `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"status"},"group_key":"status:todo","hierarchy":{"enabled":false},"parent_id":null,"page":{"cursor":"` + *pageBody.NextCursor + `"}}`,
		"parent": `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":true},"parent_id":"issue-one","page":{"cursor":"` + *pageBody.NextCursor + `"}}`,
	} {
		mismatch := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", body, headers)
		if mismatch.Code != http.StatusConflict || !containsJSON(mismatch.Body.Bytes(), `"error":"cursor_query_mismatch"`) {
			t.Fatalf("%s mismatch = %d %s", name, mismatch.Code, mismatch.Body.String())
		}
	}
	workspaceMismatch := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"page":{"cursor":"`+*pageBody.NextCursor+`"}}`, map[string]string{"Authorization": "Bearer user-two-token", "X-Workspace-Slug": "two"})
	if workspaceMismatch.Code != http.StatusConflict || !containsJSON(workspaceMismatch.Body.Bytes(), `"error":"cursor_query_mismatch"`) {
		t.Fatalf("workspace mismatch = %d %s", workspaceMismatch.Code, workspaceMismatch.Body.String())
	}
	limit := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"page":{"limit":101}}`, headers)
	if limit.Code != http.StatusBadRequest {
		t.Fatalf("limit = %d %s", limit.Code, limit.Body.String())
	}

	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET assignee_type='member',assignee_id='actor-shared' WHERE id='issue-one'; UPDATE workspace_issues SET assignee_type='agent',assignee_id='agent-one',creator_type='agent',creator_id='agent-creator' WHERE id='issue-two'; INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,assignee_type,assignee_id,creator_type,creator_id,position,metadata,properties,asset_ids,created_at,updated_at) VALUES ('issue-cross','workspace-one',4,'ONE-4','Cross pair','todo','none','member','agent-one','member','` + userOne + `',4,'{}','{}','[]','2026-08-13T00:00:05Z','2026-08-13T00:00:05Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET creator_type='member',creator_id='actor-shared' WHERE id='issue-cross'`); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET assignee_type='member',assignee_id=? WHERE id='issue-child'`, userOne); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []struct{ name, body, id string }{
		{"workspace actor types", `{"query":{"scope":{"kind":"workspace","assignee_types":["agent"]},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`, "issue-two"},
		{"my assigned", `{"query":{"scope":{"kind":"my","relation":"assigned"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`, "issue-child"},
		{"my created", `{"query":{"scope":{"kind":"my","relation":"created"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`, "issue-one"},
		{"my any", `{"query":{"scope":{"kind":"my","relation":"any"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"hierarchy":{"enabled":false}}`, "issue-one"},
	} {
		response := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", scope.body, headers)
		if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"id":"`+scope.id+`"`) {
			t.Fatalf("scope %s = %d %s", scope.name, response.Code, response.Body.String())
		}
	}
	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET assignee_type=NULL,assignee_id=NULL WHERE id='issue-child'`); err != nil {
		t.Fatal(err)
	}
	actorFacets := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/facets", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"facets":[{"kind":"assignee"},{"kind":"creator"},{"kind":"project"}]}`, headers)
	if actorFacets.Code != http.StatusOK || !containsJSON(actorFacets.Body.Bytes(), `"kind":"assignee"`, `"key":"agent:agent-one"`, `"kind":"creator"`, `"key":"agent:agent-creator"`, `"kind":"project"`, `"key":"none"`) {
		t.Fatalf("actor facets = %d %s", actorFacets.Code, actorFacets.Body.String())
	}
	mixed := runtimeRequest(runtime, http.MethodGet, "/api/issues?assignee_filters=member:actor-shared,agent:agent-one", "", headers)
	if mixed.Code != http.StatusOK || !containsJSON(mixed.Body.Bytes(), `"total":2`, `"id":"issue-one"`, `"id":"issue-two"`) || strings.Contains(mixed.Body.String(), `"id":"issue-cross"`) {
		t.Fatalf("mixed actors = %d %s", mixed.Code, mixed.Body.String())
	}
	creatorAgent := runtimeRequest(runtime, http.MethodPost, "/api/issues/query", `{"creator_filters":"agent:agent-creator"}`, headers)
	if creatorAgent.Code != http.StatusOK || !containsJSON(creatorAgent.Body.Bytes(), `"total":1`, `"id":"issue-two"`) {
		t.Fatalf("creator agent = %d %s", creatorAgent.Code, creatorAgent.Body.String())
	}
	noAssignee := runtimeRequest(runtime, http.MethodGet, "/api/issues?include_no_assignee=true", "", headers)
	if noAssignee.Code != http.StatusOK || !containsJSON(noAssignee.Body.Bytes(), `"total":1`, `"id":"issue-child"`) {
		t.Fatalf("no assignee = %d %s", noAssignee.Code, noAssignee.Body.String())
	}
	mixedPost := runtimeRequest(runtime, http.MethodPost, "/api/issues/query", `{"assignee_filters":"member:actor-shared,agent:agent-one"}`, headers)
	if mixedPost.Code != http.StatusOK || !containsJSON(mixedPost.Body.Bytes(), `"total":2`) || strings.Contains(mixedPost.Body.String(), `"id":"issue-cross"`) {
		t.Fatalf("mixed actors POST = %d %s", mixedPost.Code, mixedPost.Body.String())
	}
	if _, err := runtime.Database().Exec(`UPDATE workspace_issues SET project_id='project-one' WHERE id IN ('issue-one','issue-two')`); err != nil {
		t.Fatal(err)
	}
	noProject := runtimeRequest(runtime, http.MethodPost, "/api/issues/query", `{"include_no_project":"true"}`, headers)
	if noProject.Code != http.StatusOK || !containsJSON(noProject.Body.Bytes(), `"total":2`, `"id":"issue-child"`, `"id":"issue-cross"`) {
		t.Fatalf("no project POST = %d %s", noProject.Code, noProject.Body.String())
	}

	unknown := runtimeRequest(runtime, http.MethodPost, "/api/issues/table/rows", `{"query":{"scope":{"kind":"workspace"},"filters":{},"sort":{"field":"position","direction":"asc"}},"group":{"kind":"none"},"group_key":null,"hierarchy":{"enabled":false},"parent_id":null,"surprise":true}`, headers)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown field = %d %s", unknown.Code, unknown.Body.String())
	}
	unknownQuery := runtimeRequest(runtime, http.MethodPost, "/api/issues/query", `{"surprise":"value"}`, headers)
	if unknownQuery.Code != http.StatusBadRequest {
		t.Fatalf("unknown query field = %d %s", unknownQuery.Code, unknownQuery.Body.String())
	}
	oversized := runtimeRequest(runtime, http.MethodPost, "/api/issues/query", `{"q":"`+strings.Repeat("x", 1<<20)+`"}`, headers)
	if oversized.Code != http.StatusBadRequest {
		t.Fatalf("oversized = %d", oversized.Code)
	}
}

func TestSQLiteRuntimeAdvertisesHonestIssueCapabilities(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{Name: "backend-test", Version: "test", HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0", SQLitePath: filepath.Join(t.TempDir(), "config.db"), WorkspaceDependencies: FailClosedWorkspaceDependencies(), LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"}})
	response := runtimeRequest(runtime, http.MethodGet, "/api/config", "", nil)
	if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"issue_list":true`, `"issue_base_detail":true`, `"issue_detail_pull_requests":false`, `"issue_timeline":false`, `"issue_members":false`, `"issue_metadata":true`, `"issue_realtime":true`) {
		t.Fatalf("config = %d %s", response.Code, response.Body.String())
	}
}

func containsJSON(body []byte, fragments ...string) bool {
	encoded := string(body)
	for _, fragment := range fragments {
		if !json.Valid(body) || !contains(encoded, fragment) {
			return false
		}
	}
	return true
}

func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
