package providers

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/smallnest/goclaw/config"
)

func TestCodexAppServerProvider(t *testing.T) {
	if os.Getenv("GOCLAW_CODEX_HELPER") == "1" {
		runCodexHelper()
		os.Exit(0)
	}
	provider, err := NewCodexAppServerProvider("gpt-test", 1000, &config.ModelRuntimeConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestCodexAppServerProvider"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOCLAW_CODEX_HELPER", "1")

	response, err := provider.Chat(context.Background(),
		[]Message{{Role: "user", Content: "read the project"}},
		[]ToolDefinition{{
			Name:        "read_file",
			Description: "read a file",
			Parameters:  map[string]interface{}{"type": "object"},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].Name != "read_file" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.ToolCalls[0].ID == "" {
		t.Fatal("expected generated tool call id")
	}
	if response.Usage.TotalTokens != 15 {
		t.Fatalf("expected app-server usage, got %+v", response.Usage)
	}
}

func runCodexHelper() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		var request map[string]interface{}
		if err := decoder.Decode(&request); err != nil {
			return
		}
		id, hasID := request["id"]
		switch request["method"] {
		case "initialize":
			_ = encoder.Encode(map[string]interface{}{"id": id, "result": map[string]interface{}{}})
		case "thread/start":
			_ = encoder.Encode(map[string]interface{}{
				"id": id,
				"result": map[string]interface{}{
					"thread": map[string]interface{}{"id": "thr_test"},
				},
			})
		case "turn/start":
			_ = encoder.Encode(map[string]interface{}{
				"id":     id,
				"result": map[string]interface{}{"turn": map[string]interface{}{"id": "turn_test"}},
			})
			_ = encoder.Encode(map[string]interface{}{
				"method": "item/completed",
				"params": map[string]interface{}{
					"item": map[string]interface{}{
						"type": "agentMessage",
						"text": `{"content":"","tool_calls":[{"id":"","name":"read_file","params":{"path":"README.md"}}],"finish_reason":"tool_calls"}`,
					},
				},
			})
			_ = encoder.Encode(map[string]interface{}{
				"method": "turn/completed",
				"params": map[string]interface{}{
					"turn": map[string]interface{}{
						"status": "completed",
						"usage":  map[string]interface{}{"inputTokens": 10, "outputTokens": 5, "totalTokens": 15},
					},
				},
			})
		default:
			if hasID {
				_ = encoder.Encode(map[string]interface{}{"id": id, "result": map[string]interface{}{}})
			}
		}
	}
}
