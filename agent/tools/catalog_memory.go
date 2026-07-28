package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/smallnest/goclaw/memory/catalog"
)

// NewCatalogMemorySearchTool searches only approved catalog records. Pending,
// superseded, withdrawn, and expired records are excluded by the service.
func NewCatalogMemorySearchTool(service *catalog.Service) *BaseTool {
	return NewBaseTool(
		"search_project_memory",
		"Search approved, project-scoped catalog memory with provenance, version, authority aliases, and lifecycle warnings.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The fact, decision, constraint, preference, or prior context to find.",
				},
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Project boundary. Omit only when the configured default project is correct.",
				},
				"kinds": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []interface{}{
							"goal", "decision", "constraint", "requirement", "fact",
							"preference", "procedure", "lesson", "context", "conversation", "source",
						},
					},
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results from 1 to 20.",
					"default":     6,
				},
			},
			"required": []interface{}{"query"},
		},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			query := strings.TrimSpace(stringToolParam(params["query"]))
			if query == "" {
				return "", fmt.Errorf("query is required")
			}
			projectID, err := catalog.ResolveScopedProject(
				ctx,
				stringToolParam(params["project_id"]),
			)
			if err != nil {
				return "", err
			}
			limit := intToolParam(params["limit"], 6)
			if limit < 1 {
				limit = 1
			}
			if limit > 20 {
				limit = 20
			}
			kinds := make([]catalog.RecordKind, 0)
			for _, value := range stringSliceToolParam(params["kinds"]) {
				kinds = append(kinds, catalog.RecordKind(value))
			}
			results, err := service.Search(catalog.SearchQuery{
				Query:         query,
				ProjectID:     projectID,
				Kinds:         kinds,
				IncludeShared: true,
				Limit:         limit,
			})
			if err != nil {
				return "", fmt.Errorf("search project memory: %w", err)
			}
			type conciseResult struct {
				ID        string             `json:"id"`
				Title     string             `json:"title"`
				Kind      catalog.RecordKind `json:"kind"`
				Content   string             `json:"content"`
				Score     float64            `json:"score"`
				Citation  string             `json:"citation"`
				Warnings  []string           `json:"warnings,omitempty"`
				MatchedBy []string           `json:"matched_by,omitempty"`
			}
			output := make([]conciseResult, 0, len(results))
			for _, result := range results {
				content := result.Record.Content
				if len([]rune(content)) > 1200 {
					content = string([]rune(content)[:1200]) + "…"
				}
				output = append(output, conciseResult{
					ID:        result.Record.ID,
					Title:     result.Record.Title,
					Kind:      result.Record.Kind,
					Content:   content,
					Score:     result.Score,
					Citation:  result.Citation,
					Warnings:  result.Warnings,
					MatchedBy: result.MatchedBy,
				})
			}
			data, err := json.Marshal(output)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}

// NewCatalogMemoryProposalTool gives the model a candidate-only write path.
// Promotion to durable memory remains a separately authenticated human action.
func NewCatalogMemoryProposalTool(service *catalog.Service) *BaseTool {
	return NewBaseTool(
		"propose_project_memory",
		"Create a pending catalog-memory candidate for human review. This never makes the content retrievable as approved memory.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{"type": "string"},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Short catalog title.",
				},
				"abstract": map[string]interface{}{
					"type":        "string",
					"description": "Why this information is durable and useful.",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The complete memory claim or procedure.",
				},
				"kind": map[string]interface{}{
					"type": "string",
					"enum": []interface{}{
						"goal", "decision", "constraint", "requirement", "fact",
						"preference", "procedure", "lesson", "context", "conversation", "source",
					},
				},
				"subjects": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"type": "string"},
				},
				"source_uri": map[string]interface{}{
					"type":        "string",
					"description": "Stable source identifier, for example obsidian://project/path.md or trace:<id>.",
				},
				"evidence_refs": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"type": "string"},
				},
				"supersedes_id": map[string]interface{}{"type": "string"},
				"contradicts_ids": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"type": "string"},
				},
				"expires_at": map[string]interface{}{
					"type":        "string",
					"description": "Optional RFC3339 expiry for temporary context.",
				},
				"confidence": map[string]interface{}{
					"type":        "number",
					"description": "Evidence confidence from 0 to 1.",
				},
			},
			"required": []interface{}{"title", "abstract", "content", "kind", "source_uri"},
		},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			projectID, err := catalog.ResolveScopedProject(
				ctx,
				stringToolParam(params["project_id"]),
			)
			if err != nil {
				return "", err
			}
			relations := make([]catalog.Relation, 0)
			if target := strings.TrimSpace(stringToolParam(params["supersedes_id"])); target != "" {
				relations = append(relations, catalog.Relation{
					Type:     catalog.RelationSupersedes,
					TargetID: target,
				})
			}
			for _, target := range stringSliceToolParam(params["contradicts_ids"]) {
				relations = append(relations, catalog.Relation{
					Type:     catalog.RelationContradicts,
					TargetID: target,
				})
			}
			var expiresAt *time.Time
			if raw := strings.TrimSpace(stringToolParam(params["expires_at"])); raw != "" {
				parsed, parseErr := time.Parse(time.RFC3339, raw)
				if parseErr != nil {
					return "", fmt.Errorf("expires_at must be RFC3339: %w", parseErr)
				}
				parsed = parsed.UTC()
				expiresAt = &parsed
			}
			confidence := floatToolParam(params["confidence"], 0.5)
			record, created, err := service.CreateCandidate(catalog.CreateInput{
				ProjectID:    projectID,
				Title:        stringToolParam(params["title"]),
				Abstract:     stringToolParam(params["abstract"]),
				Content:      stringToolParam(params["content"]),
				Kind:         catalog.RecordKind(stringToolParam(params["kind"])),
				Subjects:     stringSliceToolParam(params["subjects"]),
				Relations:    relations,
				EvidenceRefs: stringSliceToolParam(params["evidence_refs"]),
				Confidence:   confidence,
				ExpiresAt:    expiresAt,
				CreatedBy:    "goclaw-agent",
				Provenance: catalog.Provenance{
					SourceURI:  stringToolParam(params["source_uri"]),
					SourceKind: "agent-proposal",
					CapturedAt: time.Now().UTC(),
					AgentID:    "goclaw-agent",
				},
			})
			if err != nil {
				return "", fmt.Errorf("create memory candidate: %w", err)
			}
			data, err := json.Marshal(map[string]interface{}{
				"id":      record.ID,
				"status":  record.Status,
				"created": created,
				"message": "Candidate recorded. It is excluded from approved memory until a human reviewer approves it.",
			})
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}

func stringToolParam(value interface{}) string {
	result, _ := value.(string)
	return strings.TrimSpace(result)
}

func stringSliceToolParam(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
		return result
	default:
		return nil
	}
}

func intToolParam(value interface{}, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func floatToolParam(value interface{}, fallback float64) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	default:
		return fallback
	}
}
