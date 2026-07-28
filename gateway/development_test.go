package gateway

import (
	"strings"
	"testing"

	"github.com/smallnest/goclaw/orchestratorlite"
)

func TestDevelopmentGatewayListsTasksAndDefaultsExecutionOff(t *testing.T) {
	service, err := orchestratorlite.NewService(orchestratorlite.Config{
		Root:     t.TempDir(),
		RepoPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetDevelopmentService(service)

	result, err := handler.registry.Call("dev.tasks", "session", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	tasks, ok := result.([]orchestratorlite.Task)
	if !ok || len(tasks) != 0 {
		t.Fatalf("unexpected task list: %#v", result)
	}

	_, err = handler.registry.Call("dev.task.run", "session", map[string]interface{}{"id": "task-test"})
	if err == nil || !strings.Contains(err.Error(), "execution is disabled") {
		t.Fatalf("expected disabled execution error, got %v", err)
	}
}
