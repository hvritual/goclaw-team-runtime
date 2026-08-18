package bootstrap

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestInstalledKnowledgeQueryHTTPVisibilityFiltersAndCursor(t *testing.T) {
	fixture := newCollaborationRuntimeFixture(t, filepath.Join(t.TempDir(), "knowledge-query.db"), "knowledge-query", "knowledge-owner@example.com")
	member := addCollaborationRuntimeMember(t, fixture, "knowledge-member@example.com", "member")
	memberHeaders := collaborationHeaders(member.Token, fixture.workspaceSlug)
	for _, statement := range []string{
		`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-exact','` + fixture.workspaceID + `','lesson','published',1,'2026-08-18T01:00:00Z','2026-08-18T03:00:00Z')`,
		`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-content','` + fixture.workspaceID + `','lesson','superseded',1,'2026-08-18T01:00:00Z','2026-08-18T02:00:00Z')`,
		`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-quarantine','` + fixture.workspaceID + `','reference','quarantined',1,'2026-08-18T01:00:00Z','2026-08-18T01:00:00Z')`,
		`INSERT INTO workspace_governed_knowledge(id,workspace_id,kind,status,current_revision,created_at,updated_at) VALUES('knowledge-foreign','workspace-foreign','lesson','published',1,'2026-08-18T01:00:00Z','2026-08-18T04:00:00Z')`,
		`INSERT INTO workspace_knowledge_revisions(knowledge_id,revision,supersedes_revision,title,content,created_by,created_at) VALUES('knowledge-exact',1,0,'Retry','body','owner','2026-08-18T01:00:00Z'),('knowledge-content',1,0,'Other','retry body','owner','2026-08-18T01:00:00Z'),('knowledge-quarantine',1,0,'Unsafe','body','owner','2026-08-18T01:00:00Z'),('knowledge-foreign',1,0,'Retry foreign','body','owner','2026-08-18T01:00:00Z')`,
		`INSERT INTO workspace_knowledge_source_refs(knowledge_id,revision,ordinal,source_type,source_id,source_revision,citation) VALUES('knowledge-exact',1,0,'acceptance_conclusion','issue-1','sha256:abc','Acceptance passed'),('knowledge-content',1,0,'retrospective','retro-1','sha256:def','Recovery review'),('knowledge-quarantine',1,0,'attachment','asset-1','sha256:bad','Untrusted attachment')`,
	} {
		if _, err := fixture.runtime.Database().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	first := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge?query=retry&status=published&status=superseded&limit=1", "", memberHeaders)
	if first.Code != http.StatusOK {
		t.Fatalf("first page = %d %s", first.Code, first.Body.String())
	}
	var page struct {
		Entries    []struct{ ID, Citation, MatchedBy string } `json:"entries"`
		NextCursor *string                                    `json:"next_cursor"`
		Total      int                                        `json:"total"`
	}
	if json.Unmarshal(first.Body.Bytes(), &page) != nil || page.Total != 2 || len(page.Entries) != 1 || page.Entries[0].ID != "knowledge-exact" || page.Entries[0].Citation != "Acceptance passed" || page.NextCursor == nil {
		t.Fatalf("first body = %s", first.Body.String())
	}
	second := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge?query=retry&status=published&status=superseded&limit=1&cursor="+*page.NextCursor, "", memberHeaders)
	if second.Code != http.StatusOK || !containsJSON(second.Body.Bytes(), `"id":"knowledge-content"`) {
		t.Fatalf("second page = %d %s", second.Code, second.Body.String())
	}
	wrongFilter := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge?query=other&status=published&status=superseded&limit=1&cursor="+*page.NextCursor, "", memberHeaders)
	if wrongFilter.Code != http.StatusBadRequest {
		t.Fatalf("cross-filter cursor = %d %s", wrongFilter.Code, wrongFilter.Body.String())
	}
	memberQuarantine := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge?status=quarantined", "", memberHeaders)
	if memberQuarantine.Code != http.StatusBadRequest {
		t.Fatalf("member quarantine = %d %s", memberQuarantine.Code, memberQuarantine.Body.String())
	}
	ownerQuarantine := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge?status=quarantined", "", fixture.headers)
	if ownerQuarantine.Code != http.StatusOK || !containsJSON(ownerQuarantine.Body.Bytes(), `"id":"knowledge-quarantine"`) {
		t.Fatalf("owner quarantine = %d %s", ownerQuarantine.Code, ownerQuarantine.Body.String())
	}
	detail := runtimeRequest(fixture.runtime, http.MethodGet, "/api/knowledge/knowledge-exact", "", memberHeaders)
	if detail.Code != http.StatusOK || !containsJSON(detail.Body.Bytes(), `"source_refs":[{"type":"acceptance_conclusion"`) {
		t.Fatalf("detail = %d %s", detail.Code, detail.Body.String())
	}
}
