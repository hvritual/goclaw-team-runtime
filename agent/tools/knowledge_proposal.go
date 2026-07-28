package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/goclaw/harness"
)

// NewKnowledgeProposalTool creates the only agent-facing write path for
// governed Obsidian knowledge. Applying the proposal remains a human action.
func NewKnowledgeProposalTool(service *harness.Service) *BaseTool {
	return NewBaseTool(
		"propose_knowledge_change",
		"Create a human-reviewable proposal for a governed Obsidian Markdown file. This never changes the target file directly.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target_path": map[string]interface{}{
					"type":        "string",
					"description": "Vault-relative Markdown path under 01-goals, 02-decisions, 03-constraints, 04-requirements, or 05-knowledge.",
				},
				"proposed_content": map[string]interface{}{
					"type":        "string",
					"description": "Complete proposed Markdown content for the target file.",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Why this change should be accepted.",
				},
				"evidence_trace_id": map[string]interface{}{
					"type":        "string",
					"description": "Optional trace that supports this proposal.",
				},
			},
			"required": []interface{}{"target_path", "proposed_content", "reason"},
		},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			target, _ := params["target_path"].(string)
			content, _ := params["proposed_content"].(string)
			reason, _ := params["reason"].(string)
			evidence, _ := params["evidence_trace_id"].(string)
			proposal, err := service.CreateKnowledgeProposal(target, content, reason, evidence, "goclaw-agent")
			if err != nil {
				return "", fmt.Errorf("create knowledge proposal: %w", err)
			}
			data, err := json.Marshal(map[string]interface{}{
				"id":          proposal.ID,
				"status":      proposal.Status,
				"target_path": proposal.TargetPath,
				"message":     "Proposal created. The governed file is unchanged until a human approves it.",
			})
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}

func NewReadKnowledgeTool(service *harness.Service) *BaseTool {
	return NewBaseTool(
		"read_project_knowledge",
		"Read one governed Markdown file from the active project's Obsidian vault. This tool is strictly read-only.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Vault-relative Markdown path under the governed project folders.",
				},
			},
			"required": []interface{}{"path"},
		},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			path, _ := params["path"].(string)
			return service.ReadKnowledge(path)
		},
	)
}

func NewSearchKnowledgeTool(service *harness.Service) *BaseTool {
	return NewBaseTool(
		"search_project_knowledge",
		"Search confirmed goals, decisions, constraints, requirements, and knowledge in the active project's Obsidian vault.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Case-insensitive text query.",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results, from 1 to 50.",
				},
			},
			"required": []interface{}{"query"},
		},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			query, _ := params["query"].(string)
			limit := 10
			if value, ok := params["limit"].(float64); ok {
				limit = int(value)
			}
			results, err := service.SearchKnowledge(query, limit)
			if err != nil {
				return "", err
			}
			data, err := json.Marshal(results)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}
