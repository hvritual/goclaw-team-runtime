package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/ouroboros"
	"github.com/smallnest/goclaw/teamcontrol"
)

func TestOuroborosGatewayRegistersControlPlaneMethods(t *testing.T) {
	service, err := ouroboros.NewService(ouroboros.Config{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetOuroborosService(service)

	result, err := handler.registry.Call(
		"ouroboros.sessions",
		"obsidian-session",
		map[string]interface{}{"project_id": "project-test"},
	)
	if err != nil {
		t.Fatal(err)
	}
	sessions, ok := result.([]ouroboros.Session)
	if !ok || len(sessions) != 0 {
		t.Fatalf("unexpected session list: %#v", result)
	}
	if _, err := handler.registry.Call(
		"ouroboros.session.compile",
		"obsidian-session",
		map[string]interface{}{"id": "ouro-test"},
	); err == nil {
		t.Fatal("compile must fail when Orchestrator Lite is disabled")
	}
}

func TestOuroborosDirectCompileFailsClosedInTeamMode(t *testing.T) {
	ouroService, err := ouroboros.NewService(ouroboros.Config{
		Root: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	development, err := dev.NewService(dev.Config{
		Root:     t.TempDir(),
		RepoPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	teamService, err := teamcontrol.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetOuroborosService(ouroService)
	handler.SetDevelopmentService(development)
	handler.SetTeamControlService(teamService)

	_, err = handler.registry.Call(
		"ouroboros.session.compile",
		teamSessionID("alice"),
		map[string]interface{}{"id": "ouro-test"},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "direct Ouroboros compilation is disabled") {
		t.Fatalf("team mode compiled directly into Orchestrator Lite: %v", err)
	}
}

func TestResolveOuroEvidenceFileRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "evidence.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := resolveOuroEvidenceFile(root, link, 1024); err == nil {
		t.Fatal("evidence symlink escaping the runtime root must be rejected")
	}
}
