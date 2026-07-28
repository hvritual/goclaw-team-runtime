package tools

import (
	"context"
	"testing"

	"github.com/smallnest/goclaw/ouroboros"
)

type ouroborosToolModel struct{}

func (ouroborosToolModel) Generate(
	_ context.Context,
	_ ouroboros.ModelRequest,
) (ouroboros.ModelResponse, error) {
	return ouroboros.ModelResponse{
		Content: `{
			"summary":"request is testable",
			"goal":{"clarity":0.95,"justification":"specific"},
			"constraint":{"clarity":0.95,"justification":"bounded"},
			"success":{"clarity":0.95,"justification":"verified"},
			"context":{"clarity":0.95,"justification":"repository supplied"},
			"questions":[],
			"assumptions":[],
			"unresolved":[],
			"decisions":[]
		}`,
		Model: "test",
	}, nil
}

func TestOuroborosChannelToolsUseConfiguredRepoAndExcludeApprovals(t *testing.T) {
	service, err := ouroboros.NewService(ouroboros.Config{
		Root:                t.TempDir(),
		RequiredReadyStreak: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.SetModel(ouroborosToolModel{})
	repoPath := t.TempDir()
	toolset := NewOuroborosTools(service, repoPath)

	var start Tool
	for _, tool := range toolset {
		switch tool.Name() {
		case "ouroboros_start":
			start = tool
		case "ouroboros_seed_approve", "ouroboros_compile", "ouroboros_execute",
			"ouroboros_evolution_approve":
			t.Fatalf("privileged control-plane tool leaked to chat: %s", tool.Name())
		}
	}
	if start == nil {
		t.Fatal("ouroboros_start tool was not registered")
	}
	if _, err := start.Execute(context.Background(), map[string]interface{}{
		"project_id":  "project-test",
		"raw_request": "Add deterministic coverage.",
		"brownfield":  true,
	}); err != nil {
		t.Fatal(err)
	}
	sessions, err := service.ListSessions("project-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].RepoPath != repoPath {
		t.Fatalf("configured repository path was not applied: %#v", sessions)
	}
}
