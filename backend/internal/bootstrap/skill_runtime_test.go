package bootstrap

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
)

func TestSQLiteRuntimeEmptySkillListReturnsJSONArray(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "empty-skills.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "empty-skills-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Empty Skills","slug":"empty-skills"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}

	listed := runtimeRequest(runtime, http.MethodGet, "/api/skills", "", collaborationHeaders(login.Token, "empty-skills"))
	if listed.Code != http.StatusOK || strings.TrimSpace(listed.Body.String()) != "[]" {
		t.Fatalf("empty list = %d %q, want 200 []", listed.Code, listed.Body.String())
	}
}

func TestSQLiteRuntimeSkillAdminCreatesFirstDraftVersion(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skills.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "skill-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skills","slug":"skills"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}

	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Release helper","description":"Ship safely"}`, collaborationHeaders(login.Token, "skills"))
	if created.Code != http.StatusCreated || !containsJSON(created.Body.Bytes(),
		`"name":"Release helper"`, `"description":"Ship safely"`, `"version":"1"`, `"status":"draft"`, `"revision":1`,
	) {
		t.Fatalf("create skill = %d %s", created.Code, created.Body.String())
	}
}

func TestSQLiteRuntimeSkillCreateBindsInitialVersionAndRejectsFiles(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-binding.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "skill-binding-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Binding","slug":"skill-binding"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(login.Token, "skill-binding")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Bound helper"}`, headers)
	var skill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil || skill.ID == "" || skill.VersionID == "" {
		t.Fatalf("create skill = %d %s", created.Code, created.Body.String())
	}
	var boundVersion string
	if err := runtime.Database().QueryRow(`SELECT skill_version_id FROM workspace_skill_bindings WHERE skill_id=?`, skill.ID).Scan(&boundVersion); err != nil || boundVersion != skill.VersionID {
		t.Fatalf("initial binding = %q, %v; want %q", boundVersion, err, skill.VersionID)
	}

	rejected := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Files are S05B","files":[{"path":"SKILL.md","content":"body"}]}`, headers)
	if rejected.Code != http.StatusServiceUnavailable || !containsJSON(rejected.Body.Bytes(), `"error":"skill files and import are unavailable"`) {
		t.Fatalf("file rejection = %d %s", rejected.Code, rejected.Body.String())
	}
	var skillCount int
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM system_skills`).Scan(&skillCount); err != nil || skillCount != 1 {
		t.Fatalf("Skill count after rejected files = %d, %v", skillCount, err)
	}
}

func TestSQLiteRuntimeSkillLifecycleCreatesImmutableVersionsAndRetainsBinding(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-lifecycle.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	login := verifyRuntimeLogin(t, runtime, "skill-lifecycle-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Lifecycle","slug":"skill-lifecycle"}`, map[string]string{
		"Authorization": "Bearer " + login.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(login.Token, "skill-lifecycle")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Release helper","description":"v1"}`, headers)
	var first struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &first) != nil || first.ID == "" || first.VersionID == "" {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	manifest := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+first.ID+"/files", `{"path":"SKILL.md","content":"# Release helper","expected_revision":1}`, headers)
	if manifest.Code != http.StatusCreated {
		t.Fatalf("add required manifest = %d %s", manifest.Code, manifest.Body.String())
	}

	versioned := runtimeRequest(runtime, http.MethodPut, "/api/skills/"+first.ID, `{"description":"v2","expected_revision":2}`, headers)
	var second struct {
		VersionID string `json:"version_id"`
	}
	if versioned.Code != http.StatusOK || json.Unmarshal(versioned.Body.Bytes(), &second) != nil || second.VersionID == "" || second.VersionID == first.VersionID || !containsJSON(versioned.Body.Bytes(), `"version":"3"`, `"description":"v2"`, `"status":"draft"`, `"revision":3`) {
		t.Fatalf("create version = %d %s", versioned.Code, versioned.Body.String())
	}
	stale := runtimeRequest(runtime, http.MethodPut, "/api/skills/"+first.ID, `{"description":"stale","expected_revision":2}`, headers)
	if stale.Code != http.StatusConflict || !containsJSON(stale.Body.Bytes(), `"code":"revision_conflict"`, `"current_revision":3`) {
		t.Fatalf("stale version = %d %s", stale.Code, stale.Body.String())
	}
	published := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+first.ID+"/versions/"+second.VersionID+"/publish", `{"expected_revision":3}`, headers)
	if published.Code != http.StatusOK || !containsJSON(published.Body.Bytes(), `"status":"published"`, `"revision":4`) {
		t.Fatalf("publish = %d %s", published.Code, published.Body.String())
	}

	historical := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+first.ID+"?version_id="+first.VersionID, "", headers)
	if historical.Code != http.StatusOK || !containsJSON(historical.Body.Bytes(), `"version":"1"`, `"description":"v1"`, `"status":"draft"`) {
		t.Fatalf("historical version = %d %s", historical.Code, historical.Body.String())
	}
	var boundVersion string
	if err := runtime.Database().QueryRow(`SELECT skill_version_id FROM workspace_skill_bindings WHERE skill_id=?`, first.ID).Scan(&boundVersion); err != nil || boundVersion != first.VersionID {
		t.Fatalf("retained binding = %q, %v; want %q", boundVersion, err, first.VersionID)
	}

	archived := runtimeRequest(runtime, http.MethodDelete, "/api/skills/"+first.ID, `{"expected_revision":4}`, headers)
	if archived.Code != http.StatusNoContent {
		t.Fatalf("archive = %d %s", archived.Code, archived.Body.String())
	}
	listed := runtimeRequest(runtime, http.MethodGet, "/api/skills", "", headers)
	if listed.Code != http.StatusOK || containsJSON(listed.Body.Bytes(), `"id":"`+first.ID+`"`) {
		t.Fatalf("archived list = %d %s", listed.Code, listed.Body.String())
	}
	retained := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+first.ID+"?version_id="+second.VersionID, "", headers)
	if retained.Code != http.StatusOK || !containsJSON(retained.Body.Bytes(), `"status":"published"`, `"revision":5`) {
		t.Fatalf("retained archived read = %d %s", retained.Code, retained.Body.String())
	}
	restored := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+first.ID+"/restore", `{"expected_revision":5}`, headers)
	if restored.Code != http.StatusOK || !containsJSON(restored.Body.Bytes(), `"revision":6`) {
		t.Fatalf("restore = %d %s", restored.Code, restored.Body.String())
	}
}

func TestSQLiteRuntimeSkillPermissionsExposeOnlyPublishedVersionsToMembers(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-permissions.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-permission-owner@example.com")
	member := verifyRuntimeLogin(t, runtime, "skill-permission-member@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Permissions","slug":"skill-permissions"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	var workspaceID string
	if err := runtime.Database().QueryRow(`SELECT id FROM workspaces WHERE slug='skill-permissions'`).Scan(&workspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Database().Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('skill-member',?,?,'member','2026-08-18T00:00:00Z')`, workspaceID, member.UserID); err != nil {
		t.Fatal(err)
	}
	ownerHeaders := collaborationHeaders(owner.Token, "skill-permissions")
	memberHeaders := collaborationHeaders(member.Token, "skill-permissions")
	draft := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Draft helper"}`, ownerHeaders)
	var draftSkill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if draft.Code != http.StatusCreated || json.Unmarshal(draft.Body.Bytes(), &draftSkill) != nil {
		t.Fatalf("create draft = %d %s", draft.Code, draft.Body.String())
	}
	publishedCreate := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Published helper"}`, ownerHeaders)
	var publishedSkill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if publishedCreate.Code != http.StatusCreated || json.Unmarshal(publishedCreate.Body.Bytes(), &publishedSkill) != nil {
		t.Fatalf("create published candidate = %d %s", publishedCreate.Code, publishedCreate.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+publishedSkill.ID+"/versions/"+publishedSkill.VersionID+"/publish", `{"expected_revision":1}`, ownerHeaders); response.Code != http.StatusOK {
		t.Fatalf("publish = %d %s", response.Code, response.Body.String())
	}

	listed := runtimeRequest(runtime, http.MethodGet, "/api/skills", "", memberHeaders)
	if listed.Code != http.StatusOK || !containsJSON(listed.Body.Bytes(), `"name":"Published helper"`) || containsJSON(listed.Body.Bytes(), `"name":"Draft helper"`) {
		t.Fatalf("member list = %d %s", listed.Code, listed.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+draftSkill.ID+"?version_id="+draftSkill.VersionID, "", memberHeaders); response.Code != http.StatusNotFound {
		t.Fatalf("member draft read = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Denied"}`, memberHeaders); response.Code != http.StatusForbidden {
		t.Fatalf("member create = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodDelete, "/api/skills/"+publishedSkill.ID, `{"expected_revision":2}`, memberHeaders); response.Code != http.StatusForbidden {
		t.Fatalf("member archive = %d %s", response.Code, response.Body.String())
	}
	history := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+publishedSkill.ID+"/history", "", ownerHeaders)
	if history.Code != http.StatusOK || !containsJSON(history.Body.Bytes(), `"origin_workspace_id":"`+workspaceID+`"`) || !containsJSON(history.Body.Bytes(), `"action":"skill.created"`) || !containsJSON(history.Body.Bytes(), `"action":"skill.published"`) {
		t.Fatalf("owner Skill history = %d %s", history.Code, history.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+publishedSkill.ID+"/history", "", memberHeaders); response.Code != http.StatusForbidden {
		t.Fatalf("member Skill history = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodDelete, "/api/skills/"+publishedSkill.ID, `{"expected_revision":2}`, ownerHeaders); response.Code != http.StatusNoContent {
		t.Fatalf("owner archive = %d %s", response.Code, response.Body.String())
	}
	listed = runtimeRequest(runtime, http.MethodGet, "/api/skills", "", memberHeaders)
	if listed.Code != http.StatusOK || containsJSON(listed.Body.Bytes(), `"name":"Published helper"`) {
		t.Fatalf("member archived list = %d %s", listed.Code, listed.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+publishedSkill.ID+"?version_id="+publishedSkill.VersionID, "", memberHeaders); response.Code != http.StatusOK {
		t.Fatalf("member exact archived reference = %d %s", response.Code, response.Body.String())
	}
}

func TestSQLiteRuntimeSkillPublishedVersionCanBeDeprecatedOnce(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-deprecate.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-deprecate-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Deprecate","slug":"skill-deprecate"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(owner.Token, "skill-deprecate")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Old helper"}`, headers)
	var skill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/versions/"+skill.VersionID+"/publish", `{"expected_revision":1}`, headers); response.Code != http.StatusOK {
		t.Fatalf("publish = %d %s", response.Code, response.Body.String())
	}
	deprecated := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/versions/"+skill.VersionID+"/deprecate", `{"expected_revision":2}`, headers)
	if deprecated.Code != http.StatusOK || !containsJSON(deprecated.Body.Bytes(), `"status":"deprecated"`, `"revision":3`) {
		t.Fatalf("deprecate = %d %s", deprecated.Code, deprecated.Body.String())
	}
	repeated := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/versions/"+skill.VersionID+"/deprecate", `{"expected_revision":3}`, headers)
	if repeated.Code != http.StatusConflict || !containsJSON(repeated.Body.Bytes(), `"error":"invalid skill transition"`) {
		t.Fatalf("repeat deprecate = %d %s", repeated.Code, repeated.Body.String())
	}
}

func TestSQLiteRuntimeSkillConcurrentVersionRequestsReturnOneRevisionConflict(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-concurrency.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-concurrency-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Concurrency","slug":"skill-concurrency"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(owner.Token, "skill-concurrency")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Concurrent helper"}`, headers)
	var skill struct {
		ID string `json:"id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil || skill.ID == "" {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	manifest := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/files", `{"path":"SKILL.md","content":"# Concurrent helper","expected_revision":1}`, headers)
	if manifest.Code != http.StatusCreated {
		t.Fatalf("add required manifest = %d %s", manifest.Code, manifest.Body.String())
	}

	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for _, description := range []string{"candidate-a", "candidate-b"} {
		description := description
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodPut, "/api/skills/"+skill.ID, `{"description":"`+description+`","expected_revision":2}`, headers)
			statuses <- response.Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent statuses = %#v, want one 200 and one 409", counts)
	}
	var revision, versionCount, auditCount int
	if err := runtime.Database().QueryRow(`SELECT revision FROM system_skills WHERE id=?`, skill.ID).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM system_skill_versions WHERE skill_id=?`, skill.ID).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM system_skill_audit WHERE skill_id=?`, skill.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if revision != 3 || versionCount != 3 || auditCount != 3 {
		t.Fatalf("revision/version/audit = %d/%d/%d, want 3/3/3", revision, versionCount, auditCount)
	}
}

func TestSQLiteRuntimeSkillConcurrentPublishRequestsReturnOneRevisionConflict(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-publish-concurrency.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-publish-concurrency-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Publish Concurrency","slug":"skill-publish-concurrency"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(owner.Token, "skill-publish-concurrency")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Publish once"}`, headers)
	var skill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil || skill.ID == "" || skill.VersionID == "" {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}

	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/versions/"+skill.VersionID+"/publish", `{"expected_revision":1}`, headers)
			statuses <- response.Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent statuses = %#v, want one 200 and one 409", counts)
	}

	start = make(chan struct{})
	statuses = make(chan int, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodDelete, "/api/skills/"+skill.ID, `{"expected_revision":2}`, headers)
			statuses <- response.Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	counts = map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusNoContent] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent archive statuses = %#v, want one 204 and one 409", counts)
	}

	start = make(chan struct{})
	statuses = make(chan int, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/restore", `{"expected_revision":3}`, headers)
			statuses <- response.Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	counts = map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent restore statuses = %#v, want one 200 and one 409", counts)
	}
}

func TestSQLiteRuntimeSkillCreateAtomicallyRollsBackAuditAndBinding(t *testing.T) {
	for _, test := range []struct {
		name    string
		trigger string
	}{
		{name: "audit", trigger: `CREATE TRIGGER reject_skill_audit BEFORE INSERT ON system_skill_audit BEGIN SELECT RAISE(ABORT,'reject audit'); END`},
		{name: "binding", trigger: `CREATE TRIGGER reject_skill_binding BEFORE INSERT ON workspace_skill_bindings BEGIN SELECT RAISE(ABORT,'reject binding'); END`},
	} {
		t.Run(test.name, func(t *testing.T) {
			runtime := newRuntimeForConfig(t, Config{
				Name: "backend-test", Version: "test",
				HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
				SQLitePath:            filepath.Join(t.TempDir(), "skill-rollback.db"),
				WorkspaceDependencies: FailClosedWorkspaceDependencies(),
				LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
			})
			owner := verifyRuntimeLogin(t, runtime, "skill-rollback-"+test.name+"@example.com")
			if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Rollback","slug":"skill-rollback"}`, map[string]string{
				"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
			}); response.Code != http.StatusCreated {
				t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
			}
			if _, err := runtime.Database().Exec(test.trigger); err != nil {
				t.Fatal(err)
			}
			response := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Must roll back"}`, collaborationHeaders(owner.Token, "skill-rollback"))
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("create with rejected %s = %d %s", test.name, response.Code, response.Body.String())
			}
			for _, table := range []string{"system_skills", "system_skill_versions", "system_skill_audit", "workspace_skill_bindings"} {
				var count int
				if err := runtime.Database().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
					t.Fatalf("%s count = %d, %v; want 0", table, count, err)
				}
			}
		})
	}
}

func TestSQLiteRuntimeSkillCatalogReopensWithVersionAndBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skill-reopen.db")
	config := Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath: path, WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth: auth.LocalAuthConfig{VerificationCode: "888888"},
	}
	runtime := newRuntimeForConfig(t, config)
	owner := verifyRuntimeLogin(t, runtime, "skill-reopen-owner@example.com")
	if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"Skill Reopen","slug":"skill-reopen"}`, map[string]string{
		"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
	}); response.Code != http.StatusCreated {
		t.Fatalf("create workspace = %d %s", response.Code, response.Body.String())
	}
	headers := collaborationHeaders(owner.Token, "skill-reopen")
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Retained helper"}`, headers)
	var skill struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodPost, "/api/skills/"+skill.ID+"/versions/"+skill.VersionID+"/publish", `{"expected_revision":1}`, headers); response.Code != http.StatusOK {
		t.Fatalf("publish = %d %s", response.Code, response.Body.String())
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := newRuntimeForConfig(t, config)
	listed := runtimeRequest(restarted, http.MethodGet, "/api/skills", "", headers)
	if listed.Code != http.StatusOK || !containsJSON(listed.Body.Bytes(), `"id":"`+skill.ID+`"`, `"version_id":"`+skill.VersionID+`"`, `"status":"published"`) {
		t.Fatalf("reopened list = %d %s", listed.Code, listed.Body.String())
	}
	var boundVersion string
	if err := restarted.Database().QueryRow(`SELECT skill_version_id FROM workspace_skill_bindings WHERE skill_id=?`, skill.ID).Scan(&boundVersion); err != nil || boundVersion != skill.VersionID {
		t.Fatalf("reopened binding = %q, %v; want %q", boundVersion, err, skill.VersionID)
	}
}

func TestSQLiteRuntimeSkillHTTPBoundariesDenyUnauthenticatedCSRFAndOtherWorkspace(t *testing.T) {
	runtime := newRuntimeForConfig(t, Config{
		Name: "backend-test", Version: "test",
		HTTPAddress: "127.0.0.1:0", GRPCAddress: "127.0.0.1:0",
		SQLitePath:            filepath.Join(t.TempDir(), "skill-boundaries.db"),
		WorkspaceDependencies: FailClosedWorkspaceDependencies(),
		LocalAuth:             auth.LocalAuthConfig{VerificationCode: "888888"},
	})
	owner := verifyRuntimeLogin(t, runtime, "skill-boundaries-owner@example.com")
	for _, workspace := range []struct{ name, slug string }{{"Workspace A", "skill-a"}, {"Workspace B", "skill-b"}} {
		if response := runtimeRequest(runtime, http.MethodPost, "/api/workspaces", `{"name":"`+workspace.name+`","slug":"`+workspace.slug+`"}`, map[string]string{
			"Authorization": "Bearer " + owner.Token, "Content-Type": "application/json",
		}); response.Code != http.StatusCreated {
			t.Fatalf("create workspace %s = %d %s", workspace.slug, response.Code, response.Body.String())
		}
	}
	created := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"Workspace A only"}`, collaborationHeaders(owner.Token, "skill-a"))
	var skill struct {
		ID string `json:"id"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &skill) != nil || skill.ID == "" {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodGet, "/api/skills", "", nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodPost, "/api/skills", `{"name":"No CSRF"}`, map[string]string{
		"Cookie": "multica_auth=" + owner.Token, "Content-Type": "application/json", "X-Workspace-Slug": "skill-a",
	}); response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF create = %d %s", response.Code, response.Body.String())
	}
	otherList := runtimeRequest(runtime, http.MethodGet, "/api/skills", "", collaborationHeaders(owner.Token, "skill-b"))
	if otherList.Code != http.StatusOK || containsJSON(otherList.Body.Bytes(), `"id":"`+skill.ID+`"`) {
		t.Fatalf("other Workspace list = %d %s", otherList.Code, otherList.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID, "", collaborationHeaders(owner.Token, "skill-b")); response.Code != http.StatusNotFound {
		t.Fatalf("other Workspace get = %d %s", response.Code, response.Body.String())
	}
	if response := runtimeRequest(runtime, http.MethodGet, "/api/skills/"+skill.ID+"/history", "", collaborationHeaders(owner.Token, "skill-b")); response.Code != http.StatusNotFound {
		t.Fatalf("other Workspace history = %d %s", response.Code, response.Body.String())
	}
}
