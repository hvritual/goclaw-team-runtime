package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteRuntimeServesIssueCatalogRoutes(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-catalog-red.db"),
		"issue-catalog-red",
		"issue-catalog-red@example.com",
	)

	for _, probe := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "label catalog", method: http.MethodGet, path: "/api/labels?resource_type=issue"},
		{name: "issue labels", method: http.MethodGet, path: "/api/issues/" + fixture.issueID + "/labels"},
		{name: "property catalog", method: http.MethodGet, path: "/api/properties"},
		{name: "acceptance conclusions", method: http.MethodGet, path: "/api/issues/" + fixture.issueID + "/acceptance-conclusions"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			response := runtimeRequest(fixture.runtime, probe.method, probe.path, probe.body, fixture.headers)
			if response.Code == http.StatusNotFound {
				t.Fatalf("Issue catalog route is missing: %s %s = %d %s", probe.method, probe.path, response.Code, response.Body.String())
			}
		})
	}

	config := runtimeRequest(fixture.runtime, http.MethodGet, "/api/config", "", nil)
	if config.Code != http.StatusOK || !containsJSON(
		config.Body.Bytes(),
		`"issue_labels":true`,
		`"issue_properties":true`,
		`"issue_acceptance":true`,
	) {
		t.Fatalf("Issue catalog capabilities = %d %s", config.Code, config.Body.String())
	}
}

func TestSQLiteRuntimeHonorsIssueCatalogContract(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "issue-catalog-contract.db")
	fixture := newCollaborationRuntimeFixture(
		t,
		databasePath,
		"issue-catalog-contract",
		"issue-catalog-owner@example.com",
	)
	member := addCollaborationRuntimeMember(t, fixture, "issue-catalog-member@example.com", "member")
	outsider := verifyRuntimeLogin(t, fixture.runtime, "issue-catalog-outsider@example.com")

	t.Run("trusted identity and bounded mutation boundary", func(t *testing.T) {
		missingAuth := runtimeRequest(fixture.runtime, http.MethodGet, "/api/labels?resource_type=issue", "", map[string]string{
			"X-Workspace-Slug": fixture.workspaceSlug,
		})
		assertRuntimeResponse(t, missingAuth.Code, missingAuth.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)

		missingWorkspace := runtimeRequest(fixture.runtime, http.MethodGet, "/api/labels?resource_type=issue", "", map[string]string{
			"Authorization": "Bearer " + fixture.login.Token,
		})
		assertRuntimeResponse(t, missingWorkspace.Code, missingWorkspace.Body.String(), http.StatusBadRequest, `{"error":"workspace is required"}`)

		foreign := runtimeRequest(fixture.runtime, http.MethodGet, "/api/labels?resource_type=issue", "", collaborationHeaders(outsider.Token, fixture.workspaceSlug))
		if foreign.Code != http.StatusNotFound {
			t.Fatalf("foreign catalog = %d %s", foreign.Code, foreign.Body.String())
		}
		if _, err := fixture.runtime.Database().Exec(`UPDATE auth_sessions SET expires_at_unix_nano=0 WHERE user_id=?`, outsider.UserID); err != nil {
			t.Fatal(err)
		}
		expired := runtimeRequest(fixture.runtime, http.MethodGet, "/api/labels?resource_type=issue", "", collaborationHeaders(outsider.Token, fixture.workspaceSlug))
		assertRuntimeResponse(t, expired.Code, expired.Body.String(), http.StatusUnauthorized, `{"error":"user not authenticated"}`)

		mismatchHeaders := collaborationHeaders(fixture.login.Token, fixture.workspaceSlug)
		mismatchHeaders["X-Workspace-ID"] = "foreign-workspace"
		mismatch := runtimeRequest(fixture.runtime, http.MethodGet, "/api/properties", "", mismatchHeaders)
		if mismatch.Code != http.StatusNotFound {
			t.Fatalf("workspace mismatch = %d %s", mismatch.Code, mismatch.Body.String())
		}

		for name, response := range map[string]*httptest.ResponseRecorder{
			"labels":       runtimeRequest(fixture.runtime, http.MethodGet, "/api/labels?resource_type=issue", "", fixture.headers),
			"issue labels": runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/labels", "", fixture.headers),
			"properties":   runtimeRequest(fixture.runtime, http.MethodGet, "/api/properties", "", fixture.headers),
			"conclusions":  runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/acceptance-conclusions", "", fixture.headers),
		} {
			expected := map[string]string{
				"labels":       `{"labels":[],"total":0}`,
				"issue labels": `{"labels":[]}`,
				"properties":   `{"properties":[],"total":0}`,
				"conclusions":  `{"acceptance_conclusions":[],"total":0}`,
			}[name]
			assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusOK, expected)
		}

		cookieWithoutCSRF := runtimeRequest(fixture.runtime, http.MethodPost, "/api/labels", `{"name":"blocked","color":"#112233"}`, map[string]string{
			"Cookie": "multica_auth=" + fixture.login.Token, "X-Workspace-Slug": fixture.workspaceSlug, "Content-Type": "application/json",
		})
		assertRuntimeResponse(t, cookieWithoutCSRF.Code, cookieWithoutCSRF.Body.String(), http.StatusForbidden, `{"error":"invalid CSRF token"}`)

		for name, body := range map[string]string{
			"unknown":   `{"name":"bad","color":"#112233","extra":true}`,
			"trailing":  `{"name":"bad","color":"#112233"}{}`,
			"oversized": `{"name":"` + strings.Repeat("x", (1<<20)+1) + `","color":"#112233"}`,
		} {
			response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/labels", body, fixture.headers)
			assertRuntimeResponse(t, response.Code, response.Body.String(), http.StatusBadRequest, `{"error":"invalid request body"}`)
			if countCatalogRows(t, fixture.runtime, "workspace_issue_labels", fixture.workspaceID) != 0 {
				t.Fatalf("%s body wrote a label", name)
			}
		}
	})

	labelOne := createRuntimeLabel(t, fixture, `{"resource_type":"issue","name":" Priority ","description":"triage","color":"ABCDEF"}`)
	labelTwo := createRuntimeLabel(t, fixture, `{"name":"Customer","description":"visible","color":"#123456"}`)
	if labelOne.Name != "Priority" || labelOne.Color != "#abcdef" || labelOne.ResourceType != "issue" {
		t.Fatalf("normalized label = %#v", labelOne)
	}
	duplicate := runtimeRequest(fixture.runtime, http.MethodPost, "/api/labels", `{"name":"priority","color":"#654321"}`, fixture.headers)
	assertRuntimeResponse(t, duplicate.Code, duplicate.Body.String(), http.StatusConflict, `{"error":"a label with that name already exists"}`)
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER collide_catalog_label_id BEFORE INSERT ON workspace_issue_labels BEGIN SELECT RAISE(ABORT,'UNIQUE constraint failed: workspace_issue_labels.id'); END`); err != nil {
		t.Fatal(err)
	}
	idCollision := runtimeRequest(fixture.runtime, http.MethodPost, "/api/labels", `{"name":"Unique name","color":"#654321"}`, fixture.headers)
	assertRuntimeResponse(t, idCollision.Code, idCollision.Body.String(), http.StatusInternalServerError, `{"error":"failed to create label"}`)
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER collide_catalog_label_id`); err != nil {
		t.Fatal(err)
	}

	cookieHeaders := map[string]string{
		"Cookie":           "multica_auth=" + fixture.login.Token + "; multica_csrf=" + fixture.login.CSRF,
		"X-CSRF-Token":     fixture.login.CSRF,
		"X-Workspace-Slug": fixture.workspaceSlug,
		"Content-Type":     "application/json",
	}
	attachOne := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/labels", `{"label_id":"`+labelOne.ID+`"}`, cookieHeaders)
	assertLabelBag(t, attachOne, labelOne.ID)
	attachTwo := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.identifier+"/labels", `{"label_id":"`+labelTwo.ID+`"}`, fixture.headers)
	assertLabelBag(t, attachTwo, labelTwo.ID, labelOne.ID)
	idempotentAttach := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/labels", `{"label_id":"`+labelOne.ID+`"}`, fixture.headers)
	assertLabelBag(t, idempotentAttach, labelTwo.ID, labelOne.ID)
	labelCatalog := runtimeRequest(fixture.runtime, http.MethodGet, "/api/labels?resource_type=issue", "", fixture.headers)
	if labelCatalog.Code != http.StatusOK || !containsJSON(labelCatalog.Body.Bytes(), `"total":2`, `"usage_count":1`) {
		t.Fatalf("label catalog = %d %s", labelCatalog.Code, labelCatalog.Body.String())
	}
	updatedLabel := runtimeRequest(fixture.runtime, http.MethodPut, "/api/labels/"+labelTwo.ID, `{"description":"updated","color":"654321"}`, fixture.headers)
	if updatedLabel.Code != http.StatusOK || !containsJSON(updatedLabel.Body.Bytes(), `"description":"updated"`, `"color":"#654321"`) {
		t.Fatalf("update label = %d %s", updatedLabel.Code, updatedLabel.Body.String())
	}

	foreignWorkspace := runtimeRequest(fixture.runtime, http.MethodPost, "/api/workspaces", `{"name":"Catalog Foreign","slug":"issue-catalog-foreign"}`, map[string]string{
		"Authorization": "Bearer " + fixture.login.Token, "Content-Type": "application/json",
	})
	if foreignWorkspace.Code != http.StatusCreated {
		t.Fatalf("create foreign Workspace = %d %s", foreignWorkspace.Code, foreignWorkspace.Body.String())
	}
	foreignHeaders := collaborationHeaders(fixture.login.Token, "issue-catalog-foreign")
	foreignIssue := createRuntimeIssue(t, fixture.runtime, foreignHeaders, "Foreign catalog Issue")
	crossWorkspaceIssue := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+foreignIssue.ID+"/labels", "", fixture.headers)
	if crossWorkspaceIssue.Code != http.StatusNotFound {
		t.Fatalf("cross-Workspace Issue catalog = %d %s", crossWorkspaceIssue.Code, crossWorkspaceIssue.Body.String())
	}

	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_catalog_label_delete BEFORE DELETE ON workspace_issue_labels BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	blockedDelete := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/labels/"+labelOne.ID, "", fixture.headers)
	if blockedDelete.Code != http.StatusInternalServerError {
		t.Fatalf("blocked label delete = %d %s", blockedDelete.Code, blockedDelete.Body.String())
	}
	assertLabelBag(t, runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/labels", "", fixture.headers), labelTwo.ID, labelOne.ID)
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_catalog_label_delete`); err != nil {
		t.Fatal(err)
	}

	memberHeaders := collaborationHeaders(member.Token, fixture.workspaceSlug)
	memberProperty := runtimeRequest(fixture.runtime, http.MethodPost, "/api/properties", `{"name":"Denied","type":"text"}`, memberHeaders)
	assertRuntimeResponse(t, memberProperty.Code, memberProperty.Body.String(), http.StatusForbidden, `{"error":"insufficient permissions"}`)

	properties := []struct {
		name, propertyType, config, invalid, valid string
	}{
		{name: "Summary", propertyType: "text", invalid: `false`, valid: `"ready"`},
		{name: "Estimate", propertyType: "number", invalid: `"2"`, valid: `2.5`},
		{name: "Stage Choice", propertyType: "select", config: `,"config":{"options":[{"id":"option-a","name":"A","color":"#111111"},{"id":"option-b","name":"B","color":"#222222"}]}`, invalid: `"missing"`, valid: `"option-a"`},
		{name: "Areas", propertyType: "multi_select", config: `,"config":{"options":[{"id":"area-a","name":"A","color":"#333333"},{"id":"area-b","name":"B","color":"#444444"}]}`, invalid: `["missing"]`, valid: `["area-b","area-a","area-b"]`},
		{name: "Target Date", propertyType: "date", invalid: `"2026-02-30"`, valid: `"2026-08-15"`},
		{name: "Verified", propertyType: "checkbox", invalid: `"true"`, valid: `true`},
		{name: "Reference", propertyType: "url", invalid: `"javascript:alert(1)"`, valid: `"https://example.com/evidence"`},
	}
	propertyIDs := make(map[string]string, len(properties))
	for _, property := range properties {
		created := createRuntimeProperty(t, fixture, `{"name":`+quoteJSON(property.name)+`,"type":`+quoteJSON(property.propertyType)+property.config+`}`)
		propertyIDs[property.propertyType] = created.ID
		invalid := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+created.ID, `{"value":`+property.invalid+`}`, fixture.headers)
		assertRuntimeResponse(t, invalid.Code, invalid.Body.String(), http.StatusBadRequest, `{"error":"invalid request"}`)
		valid := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+created.ID, `{"value":`+property.valid+`}`, fixture.headers)
		if valid.Code != http.StatusOK || !containsJSON(valid.Body.Bytes(), `"`+created.ID+`":`) {
			t.Fatalf("set %s property = %d %s", property.propertyType, valid.Code, valid.Body.String())
		}
	}
	duplicateProperty := runtimeRequest(fixture.runtime, http.MethodPost, "/api/properties", `{"name":"estimate","type":"number"}`, fixture.headers)
	assertRuntimeResponse(t, duplicateProperty.Code, duplicateProperty.Body.String(), http.StatusConflict, `{"error":"a property with that name already exists"}`)
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER collide_catalog_property_id BEFORE INSERT ON workspace_issue_property_definitions BEGIN SELECT RAISE(ABORT,'UNIQUE constraint failed: workspace_issue_property_definitions.id'); END`); err != nil {
		t.Fatal(err)
	}
	propertyIDCollision := runtimeRequest(fixture.runtime, http.MethodPost, "/api/properties", `{"name":"Unique property","type":"text"}`, fixture.headers)
	assertRuntimeResponse(t, propertyIDCollision.Code, propertyIDCollision.Body.String(), http.StatusInternalServerError, `{"error":"failed to create property"}`)
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER collide_catalog_property_id`); err != nil {
		t.Fatal(err)
	}
	propertyCatalog := runtimeRequest(fixture.runtime, http.MethodGet, "/api/properties", "", fixture.headers)
	if propertyCatalog.Code != http.StatusOK || !containsJSON(propertyCatalog.Body.Bytes(), `"total":7`, `"usage_count":1`) {
		t.Fatalf("property catalog = %d %s", propertyCatalog.Code, propertyCatalog.Body.String())
	}
	removeUsedOption := runtimeRequest(fixture.runtime, http.MethodPatch, "/api/properties/"+propertyIDs["select"], `{"config":{"options":[{"id":"option-b","name":"B","color":"#222222"}]}}`, fixture.headers)
	assertRuntimeResponse(t, removeUsedOption.Code, removeUsedOption.Body.String(), http.StatusConflict, `{"error":"cannot remove options still in use: \"A\" (1 issue); clear or change those values first"}`)

	propertyBag := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if propertyBag.Code != http.StatusOK || !containsJSON(propertyBag.Body.Bytes(),
		`"`+propertyIDs["text"]+`":"ready"`,
		`"`+propertyIDs["number"]+`":2.5`,
		`"`+propertyIDs["multi_select"]+`":["area-a","area-b"]`,
	) {
		t.Fatalf("Issue property bag = %d %s", propertyBag.Code, propertyBag.Body.String())
	}

	archive := runtimeRequest(fixture.runtime, http.MethodPatch, "/api/properties/"+propertyIDs["text"], `{"archived":true}`, fixture.headers)
	if archive.Code != http.StatusOK || !containsJSON(archive.Body.Bytes(), `"archived":true`) {
		t.Fatalf("archive property = %d %s", archive.Code, archive.Body.String())
	}
	activeCatalog := runtimeRequest(fixture.runtime, http.MethodGet, "/api/properties", "", fixture.headers)
	archivedCatalog := runtimeRequest(fixture.runtime, http.MethodGet, "/api/properties?include_archived=true", "", fixture.headers)
	if activeCatalog.Code != http.StatusOK || !containsJSON(activeCatalog.Body.Bytes(), `"total":6`) || containsJSON(activeCatalog.Body.Bytes(), `"id":"`+propertyIDs["text"]+`"`) {
		t.Fatalf("active property catalog = %d %s", activeCatalog.Code, activeCatalog.Body.String())
	}
	if archivedCatalog.Code != http.StatusOK || !containsJSON(archivedCatalog.Body.Bytes(), `"total":7`, `"id":"`+propertyIDs["text"]+`"`, `"archived":true`) {
		t.Fatalf("archived property catalog = %d %s", archivedCatalog.Code, archivedCatalog.Body.String())
	}
	archivedSet := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+propertyIDs["text"], `{"value":"blocked"}`, fixture.headers)
	assertRuntimeResponse(t, archivedSet.Code, archivedSet.Body.String(), http.StatusBadRequest, `{"error":"invalid request"}`)
	unsetArchived := runtimeRequest(fixture.runtime, http.MethodDelete, "/api/issues/"+fixture.issueID+"/properties/"+propertyIDs["text"], "", fixture.headers)
	if unsetArchived.Code != http.StatusOK || containsJSON(unsetArchived.Body.Bytes(), `"`+propertyIDs["text"]+`":`) {
		t.Fatalf("unset archived property = %d %s", unsetArchived.Code, unsetArchived.Body.String())
	}

	tooEarly := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/acceptance-conclusions", `{"result":"accepted","rationale":"too early","evidence_refs":[]}`, fixture.headers)
	assertRuntimeResponse(t, tooEarly.Code, tooEarly.Body.String(), http.StatusConflict, `{"error":"issue must be done before recording an acceptance conclusion"}`)
	completed := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.identifier, `{"status":"done","acceptance_conclusion":{"result":"accepted","rationale":"all checks passed","evidence_refs":["trace://one"]}}`, fixture.headers)
	if completed.Code != http.StatusOK || !containsJSON(completed.Body.Bytes(), `"status":"done"`) {
		t.Fatalf("complete Issue with acceptance = %d %s", completed.Code, completed.Body.String())
	}
	secondConclusion := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/acceptance-conclusions", `{"result":"conditional","rationale":"retain follow-up","evidence_refs":["trace://two"]}`, fixture.headers)
	if secondConclusion.Code != http.StatusCreated || !containsJSON(secondConclusion.Body.Bytes(), `"result":"conditional"`, `"evidence_refs":["trace://two"]`) {
		t.Fatalf("record conclusion = %d %s", secondConclusion.Code, secondConclusion.Body.String())
	}
	invalidConclusion := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/acceptance-conclusions", `{"result":"maybe","rationale":"invalid","evidence_refs":[]}`, fixture.headers)
	assertRuntimeResponse(t, invalidConclusion.Code, invalidConclusion.Body.String(), http.StatusBadRequest, `{"error":"invalid request"}`)
	conclusions := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID+"/acceptance-conclusions", "", fixture.headers)
	if conclusions.Code != http.StatusOK || !containsJSON(conclusions.Body.Bytes(), `"total":2`, `"result":"accepted"`, `"result":"conditional"`) {
		t.Fatalf("acceptance conclusions = %d %s", conclusions.Code, conclusions.Body.String())
	}
	if countCatalogRows(t, fixture.runtime, "workspace_acceptance_knowledge_proposals", fixture.workspaceID) != 2 {
		t.Fatal("acceptance conclusions did not capture two knowledge proposals")
	}
	var sourceRevision, captureContent string
	if err := fixture.runtime.Database().QueryRow(`SELECT source_revision,content FROM workspace_acceptance_knowledge_proposals WHERE workspace_id=? AND issue_id=? ORDER BY created_at LIMIT 1`, fixture.workspaceID, fixture.issueID).Scan(&sourceRevision, &captureContent); err != nil || !strings.Contains(sourceRevision, "@sha256:") || !strings.Contains(captureContent, "all checks passed") {
		t.Fatalf("acceptance capture revision=%q content=%q err=%v", sourceRevision, captureContent, err)
	}

	rollbackIssue := createRuntimeIssue(t, fixture.runtime, fixture.headers, "Acceptance capture rollback")
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_acceptance_capture BEFORE INSERT ON workspace_acceptance_knowledge_proposals BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	failedCompletion := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+rollbackIssue.ID, `{"status":"done","acceptance_conclusion":{"result":"accepted","rationale":"must roll back","evidence_refs":[]}}`, fixture.headers)
	if failedCompletion.Code != http.StatusInternalServerError {
		t.Fatalf("failed acceptance capture = %d %s", failedCompletion.Code, failedCompletion.Body.String())
	}
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_acceptance_capture`); err != nil {
		t.Fatal(err)
	}
	rolledBack := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+rollbackIssue.ID, "", fixture.headers)
	if rolledBack.Code != http.StatusOK || !containsJSON(rolledBack.Body.Bytes(), `"status":"todo"`) {
		t.Fatalf("acceptance capture rollback = %d %s", rolledBack.Code, rolledBack.Body.String())
	}
	var rolledBackConclusions, rolledBackCaptures int
	if err := fixture.runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_issue_acceptance_conclusions WHERE workspace_id=? AND issue_id=?`, fixture.workspaceID, rollbackIssue.ID).Scan(&rolledBackConclusions); err != nil {
		t.Fatal(err)
	}
	if err := fixture.runtime.Database().QueryRow(`SELECT COUNT(*) FROM workspace_acceptance_knowledge_proposals WHERE workspace_id=? AND issue_id=?`, fixture.workspaceID, rollbackIssue.ID).Scan(&rolledBackCaptures); err != nil {
		t.Fatal(err)
	}
	if rolledBackConclusions != 0 || rolledBackCaptures != 0 {
		t.Fatalf("acceptance rollback conclusions=%d captures=%d", rolledBackConclusions, rolledBackCaptures)
	}

	if err := fixture.runtime.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := newRuntimeForConfig(t, fixture.config)
	assertLabelBag(t, runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.issueID+"/labels", "", fixture.headers), labelTwo.ID, labelOne.ID)
	retainedProperties := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if retainedProperties.Code != http.StatusOK || !containsJSON(retainedProperties.Body.Bytes(), `"`+propertyIDs["number"]+`":2.5`) {
		t.Fatalf("retained Issue properties = %d %s", retainedProperties.Code, retainedProperties.Body.String())
	}
	retainedConclusions := runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.issueID+"/acceptance-conclusions", "", fixture.headers)
	if retainedConclusions.Code != http.StatusOK || !containsJSON(retainedConclusions.Body.Bytes(), `"total":2`) {
		t.Fatalf("retained conclusions = %d %s", retainedConclusions.Code, retainedConclusions.Body.String())
	}

	deleteLabel := runtimeRequest(restarted, http.MethodDelete, "/api/labels/"+labelOne.ID, "", fixture.headers)
	assertRuntimeResponse(t, deleteLabel.Code, deleteLabel.Body.String(), http.StatusNoContent, "")
	assertLabelBag(t, runtimeRequest(restarted, http.MethodGet, "/api/issues/"+fixture.issueID+"/labels", "", fixture.headers), labelTwo.ID)
	deleteIssue := runtimeRequest(restarted, http.MethodDelete, "/api/issues/"+fixture.issueID, "", fixture.headers)
	assertRuntimeResponse(t, deleteIssue.Code, deleteIssue.Body.String(), http.StatusNoContent, "")
	for table := range map[string]struct{}{
		"workspace_issue_label_assignments":        {},
		"workspace_issue_acceptance_conclusions":   {},
		"workspace_acceptance_knowledge_proposals": {},
	} {
		if countCatalogRows(t, restarted, table, fixture.workspaceID) != 0 {
			t.Fatalf("%s survived Issue deletion", table)
		}
	}
}

func TestSQLiteRuntimeSerializesIssueCatalogWrites(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-catalog-concurrency.db"),
		"issue-catalog-concurrency",
		"issue-catalog-concurrency@example.com",
	)

	gate := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() {
			<-gate
			responses <- runtimeRequest(fixture.runtime, http.MethodPost, "/api/labels", `{"name":"Concurrent","color":"#112233"}`, fixture.headers)
		}()
	}
	close(gate)
	created, conflict := 0, 0
	for range 2 {
		switch response := <-responses; response.Code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("concurrent label = %d %s", response.Code, response.Body.String())
		}
	}
	if created != 1 || conflict != 1 || countCatalogRows(t, fixture.runtime, "workspace_issue_labels", fixture.workspaceID) != 1 {
		t.Fatalf("concurrent labels created=%d conflict=%d", created, conflict)
	}

	propertyOne := createRuntimeProperty(t, fixture, `{"name":"Concurrent A","type":"text"}`)
	propertyTwo := createRuntimeProperty(t, fixture, `{"name":"Concurrent B","type":"number"}`)
	propertyGate := make(chan struct{})
	propertyResponses := make(chan *httptest.ResponseRecorder, 2)
	for _, request := range []struct {
		id, value string
	}{{propertyOne.ID, `"one"`}, {propertyTwo.ID, `2.25`}} {
		request := request
		go func() {
			<-propertyGate
			propertyResponses <- runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+request.id, `{"value":`+request.value+`}`, fixture.headers)
		}()
	}
	close(propertyGate)
	for range 2 {
		response := <-propertyResponses
		if response.Code != http.StatusOK {
			t.Fatalf("concurrent property = %d %s", response.Code, response.Body.String())
		}
	}
	detail := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if detail.Code != http.StatusOK || !containsJSON(detail.Body.Bytes(), `"`+propertyOne.ID+`":"one"`, `"`+propertyTwo.ID+`":2.25`) {
		t.Fatalf("concurrent property bag = %d %s", detail.Code, detail.Body.String())
	}
}

func TestSQLiteRuntimeRepairsMalformedIssuePropertyBags(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-catalog-malformed-bag.db"),
		"issue-catalog-malformed-bag",
		"issue-catalog-malformed-bag@example.com",
	)
	property := createRuntimeProperty(t, fixture, `{"name":"Repair","type":"text"}`)
	if _, err := fixture.runtime.Database().Exec(
		`UPDATE workspace_issues SET properties=? WHERE workspace_id=? AND id=?`,
		`{"poison":{"nested":true}} trailing`, fixture.workspaceID, fixture.issueID,
	); err != nil {
		t.Fatalf("seed malformed property bag: %v", err)
	}

	response := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+property.ID, `{"value":"repaired"}`, fixture.headers)
	if response.Code != http.StatusOK || !containsJSON(response.Body.Bytes(), `"`+property.ID+`":"repaired"`) || strings.Contains(response.Body.String(), "poison") {
		t.Fatalf("repaired property bag = %d %s", response.Code, response.Body.String())
	}
}

func TestSQLiteRuntimeEnforcesActiveIssuePropertyLimit(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-catalog-property-limit.db"),
		"issue-catalog-property-limit",
		"issue-catalog-property-limit@example.com",
	)
	properties := make([]runtimeCatalogProperty, 0, 20)
	for index := range 20 {
		properties = append(properties, createRuntimeProperty(t, fixture, `{"name":"Limit `+string(rune('A'+index))+`","type":"text"}`))
	}
	limit := runtimeRequest(fixture.runtime, http.MethodPost, "/api/properties", `{"name":"Limit overflow","type":"text"}`, fixture.headers)
	assertRuntimeResponse(t, limit.Code, limit.Body.String(), http.StatusBadRequest, `{"error":"a workspace cannot have more than 20 active properties; archive unused ones first"}`)

	archive := runtimeRequest(fixture.runtime, http.MethodPatch, "/api/properties/"+properties[0].ID, `{"archived":true}`, fixture.headers)
	if archive.Code != http.StatusOK {
		t.Fatalf("archive property below limit = %d %s", archive.Code, archive.Body.String())
	}
	createRuntimeProperty(t, fixture, `{"name":"Limit replacement","type":"text"}`)
}

func TestSQLiteRuntimePublishesIssueCatalogEventsAfterCommit(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(
		t,
		filepath.Join(t.TempDir(), "issue-catalog-events.db"),
		"issue-catalog-events",
		"issue-catalog-events@example.com",
	)
	server := httptest.NewServer(fixture.runtime.HTTPServer())
	defer server.Close()
	socket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	defer socket.Close()

	label := createRuntimeLabel(t, fixture, `{"name":"Realtime","color":"#abcdef"}`)
	assertRealtimeEvent(t, socket, "label:created", `"id":"`+label.ID+`"`)
	attached := runtimeRequest(fixture.runtime, http.MethodPost, "/api/issues/"+fixture.issueID+"/labels", `{"label_id":"`+label.ID+`"}`, fixture.headers)
	assertLabelBag(t, attached, label.ID)
	assertRealtimeEvent(t, socket, "issue_labels:changed", `"issue_id":"`+fixture.issueID+`"`)

	property := createRuntimeProperty(t, fixture, `{"name":"Realtime property","type":"checkbox"}`)
	assertRealtimeEvent(t, socket, "property:created", `"id":"`+property.ID+`"`)
	set := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+property.ID, `{"value":true}`, fixture.headers)
	if set.Code != http.StatusOK {
		t.Fatalf("set realtime property = %d %s", set.Code, set.Body.String())
	}
	assertRealtimeEvent(t, socket, "issue_properties:changed", `"`+property.ID+`":true`)

	completed := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID, `{"status":"done","acceptance_conclusion":{"result":"accepted","rationale":"realtime complete","evidence_refs":[]}}`, fixture.headers)
	if completed.Code != http.StatusOK {
		t.Fatalf("complete realtime Issue = %d %s", completed.Code, completed.Body.String())
	}
	assertRealtimeEvent(t, socket, "issue:updated", `"status_changed":true`)

	failureSocket := dialTokenRealtime(t, server.URL, fixture.workspaceSlug, fixture.login.Token)
	defer failureSocket.Close()
	if _, err := fixture.runtime.Database().Exec(`CREATE TRIGGER block_catalog_property_write BEFORE UPDATE OF properties ON workspace_issues BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	failed := runtimeRequest(fixture.runtime, http.MethodPut, "/api/issues/"+fixture.issueID+"/properties/"+property.ID, `{"value":false}`, fixture.headers)
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("failed property write = %d %s", failed.Code, failed.Body.String())
	}
	assertNoRealtimeEvent(t, failureSocket)
	if _, err := fixture.runtime.Database().Exec(`DROP TRIGGER block_catalog_property_write`); err != nil {
		t.Fatal(err)
	}
	detail := runtimeRequest(fixture.runtime, http.MethodGet, "/api/issues/"+fixture.issueID, "", fixture.headers)
	if detail.Code != http.StatusOK || !containsJSON(detail.Body.Bytes(), `"`+property.ID+`":true`) {
		t.Fatalf("rolled-back property = %d %s", detail.Code, detail.Body.String())
	}
}

type runtimeCatalogLabel struct {
	ID           string `json:"id"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	Color        string `json:"color"`
}

type runtimeCatalogProperty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func createRuntimeLabel(t *testing.T, fixture collaborationRuntimeFixture, body string) runtimeCatalogLabel {
	t.Helper()
	response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/labels", body, fixture.headers)
	var value runtimeCatalogLabel
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &value) != nil || value.ID == "" {
		t.Fatalf("create label = %d %s", response.Code, response.Body.String())
	}
	return value
}

func createRuntimeProperty(t *testing.T, fixture collaborationRuntimeFixture, body string) runtimeCatalogProperty {
	t.Helper()
	response := runtimeRequest(fixture.runtime, http.MethodPost, "/api/properties", body, fixture.headers)
	var value runtimeCatalogProperty
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &value) != nil || value.ID == "" {
		t.Fatalf("create property = %d %s", response.Code, response.Body.String())
	}
	return value
}

func assertLabelBag(t *testing.T, response *httptest.ResponseRecorder, expected ...string) {
	t.Helper()
	var body struct {
		Labels []struct {
			ID string `json:"id"`
		} `json:"labels"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body.Labels) != len(expected) {
		t.Fatalf("label bag = %d %s, want ids=%v", response.Code, response.Body.String(), expected)
	}
	for index := range expected {
		if body.Labels[index].ID != expected[index] {
			t.Fatalf("label bag = %d %s, want ids=%v", response.Code, response.Body.String(), expected)
		}
	}
}

func countCatalogRows(t *testing.T, runtime *Runtime, table, workspaceID string) int {
	t.Helper()
	var count int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE workspace_id=?`, workspaceID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func quoteJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
