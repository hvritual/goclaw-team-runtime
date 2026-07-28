package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smallnest/goclaw/ouroboros"
)

// NewOuroborosTools exposes only interview and crystallization operations to
// conversational channels such as Feishu. Seed approval, evolution approval,
// compilation, execution, and Harness promotion remain human control-plane
// operations through Obsidian or the local CLI.
func NewOuroborosTools(service *ouroboros.Service, defaultRepoPaths ...string) []Tool {
	if service == nil {
		return nil
	}
	defaultRepoPath := ""
	if len(defaultRepoPaths) > 0 {
		defaultRepoPath = defaultRepoPaths[0]
	}
	return []Tool{
		NewBaseTool(
			"ouroboros_start",
			"Start a project-scoped Socratic requirement interview. This creates no development task and performs no code changes.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id":      map[string]interface{}{"type": "string"},
					"topic_id":        map[string]interface{}{"type": "string"},
					"title":           map[string]interface{}{"type": "string"},
					"raw_request":     map[string]interface{}{"type": "string"},
					"repo_path":       map[string]interface{}{"type": "string"},
					"base_ref":        map[string]interface{}{"type": "string"},
					"context_summary": map[string]interface{}{"type": "string"},
					"brownfield":      map[string]interface{}{"type": "boolean"},
				},
				"required": []string{"project_id", "raw_request"},
			},
			func(ctx context.Context, params map[string]interface{}) (string, error) {
				var request ouroboros.StartRequest
				if err := decodeToolParams(params, &request); err != nil {
					return "", err
				}
				if request.RepoPath == "" {
					request.RepoPath = defaultRepoPath
				}
				request.CreatedBy = "channel-human"
				session, err := service.Start(ctx, request)
				if err != nil {
					return "", err
				}
				return prettyToolJSON(session)
			},
		),
		NewBaseTool(
			"ouroboros_answer",
			"Record explicit human answers for the active Ouroboros interview and recalculate ambiguity. Never infer or fabricate an answer.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
					"answers": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"question_id": map[string]interface{}{"type": "string"},
								"text":        map[string]interface{}{"type": "string"},
							},
							"required": []string{"question_id", "text"},
						},
					},
				},
				"required": []string{"session_id", "answers"},
			},
			func(ctx context.Context, params map[string]interface{}) (string, error) {
				id, _ := params["session_id"].(string)
				if id == "" {
					return "", errors.New("session_id is required")
				}
				var payload struct {
					Answers []ouroboros.Answer `json:"answers"`
				}
				if err := decodeToolParams(params, &payload); err != nil {
					return "", err
				}
				session, err := service.Answer(ctx, id, ouroboros.AnswerRequest{
					Answers:  payload.Answers,
					Actor:    "channel-human",
					Reassess: true,
				})
				if err != nil {
					return "", err
				}
				return prettyToolJSON(session)
			},
		),
		NewBaseTool(
			"ouroboros_get",
			"Read one Ouroboros interview, ambiguity score, immutable Seed references, evaluation, and evolution status.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
			func(_ context.Context, params map[string]interface{}) (string, error) {
				id, _ := params["session_id"].(string)
				if id == "" {
					return "", errors.New("session_id is required")
				}
				session, err := service.GetSession(id)
				if err != nil {
					return "", err
				}
				return prettyToolJSON(session)
			},
		),
		NewBaseTool(
			"ouroboros_reassess",
			"Run another independent ambiguity assessment after the human has supplied enough facts. It cannot approve or compile a Seed.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
			func(ctx context.Context, params map[string]interface{}) (string, error) {
				id, _ := params["session_id"].(string)
				if id == "" {
					return "", errors.New("session_id is required")
				}
				session, err := service.Reassess(ctx, id, "channel-human")
				if err != nil {
					return "", err
				}
				return prettyToolJSON(session)
			},
		),
		NewBaseTool(
			"ouroboros_crystallize",
			"Crystallize a seed-ready interview into an immutable Seed proposal. Human approval is still required in Obsidian or the local CLI before task compilation.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
			func(ctx context.Context, params map[string]interface{}) (string, error) {
				id, _ := params["session_id"].(string)
				if id == "" {
					return "", errors.New("session_id is required")
				}
				session, err := service.Crystallize(ctx, id, "channel-human")
				if err != nil {
					return "", err
				}
				return prettyToolJSON(session)
			},
		),
	}
}

func decodeToolParams(params map[string]interface{}, target any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid tool parameters: %w", err)
	}
	return nil
}

func prettyToolJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
