package gateway

import (
	"fmt"
	"strings"
	"time"

	"github.com/smallnest/goclaw/governance"
	"github.com/smallnest/goclaw/memory/catalog"
)

func (h *Handler) registerMemoryCatalogMethods() {
	h.registry.Register("memory.catalog.status", func(_ string, params map[string]interface{}) (interface{}, error) {
		return h.catalogSvc.Stats(stringParam(params["project_id"]))
	})

	h.registry.Register("memory.catalog.list", func(_ string, params map[string]interface{}) (interface{}, error) {
		limit := integerParam(params["limit"], 200)
		return h.catalogSvc.List(
			stringParam(params["project_id"]),
			catalog.RecordStatus(stringParam(params["status"])),
			limit,
		)
	})

	h.registry.Register("memory.catalog.get", func(_ string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		return h.catalogSvc.Get(id)
	})

	h.registry.Register("memory.catalog.search", func(_ string, params map[string]interface{}) (interface{}, error) {
		kinds := make([]catalog.RecordKind, 0)
		for _, value := range stringSliceParam(params["kinds"]) {
			kinds = append(kinds, catalog.RecordKind(value))
		}
		statuses := make([]catalog.RecordStatus, 0)
		for _, value := range stringSliceParam(params["statuses"]) {
			statuses = append(statuses, catalog.RecordStatus(value))
		}
		includeShared, _ := params["include_shared"].(bool)
		includeExpired, _ := params["include_expired"].(bool)
		return h.catalogSvc.Search(catalog.SearchQuery{
			Query:          stringParam(params["query"]),
			ProjectID:      stringParam(params["project_id"]),
			Statuses:       statuses,
			Kinds:          kinds,
			Facets:         stringSlicesMapParam(params["facets"]),
			AuthorityIDs:   stringSliceParam(params["authority_ids"]),
			IncludeShared:  includeShared,
			IncludeExpired: includeExpired,
			Limit:          integerParam(params["limit"], 20),
		})
	})

	h.registry.Register("memory.catalog.candidate.create", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		title := stringParam(params["title"])
		content := stringParam(params["content"])
		kind := catalog.RecordKind(stringParam(params["kind"]))
		if title == "" || content == "" || kind == "" {
			return nil, fmt.Errorf("title, content, and kind are required")
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		relations := make([]catalog.Relation, 0)
		if target := stringParam(params["supersedes_id"]); target != "" {
			relations = append(relations, catalog.Relation{
				Type:     catalog.RelationSupersedes,
				TargetID: target,
			})
		}
		for _, target := range stringSliceParam(params["contradicts_ids"]) {
			relations = append(relations, catalog.Relation{
				Type:     catalog.RelationContradicts,
				TargetID: target,
			})
		}
		record, created, err := h.catalogSvc.CreateCandidate(catalog.CreateInput{
			ProjectID:    stringParam(params["project_id"]),
			Collection:   stringParam(params["collection"]),
			Title:        title,
			Abstract:     stringParam(params["abstract"]),
			Content:      content,
			Kind:         kind,
			Language:     stringParam(params["language"]),
			Subjects:     stringSliceParam(params["subjects"]),
			Facets:       stringSlicesMapParam(params["facets"]),
			AuthorityIDs: stringSliceParam(params["authority_ids"]),
			Relations:    relations,
			EvidenceRefs: stringSliceParam(params["evidence_refs"]),
			Confidence:   numberParam(params["confidence"], 0.5),
			ExpiresAt:    timeParam(params["expires_at"]),
			CreatedBy:    actor,
			Provenance: catalog.Provenance{
				SourceURI:  stringParam(params["source_uri"]),
				SourceKind: "gateway-proposal",
				CapturedAt: time.Now().UTC(),
				AgentID:    actor,
				TraceID:    stringParam(params["trace_id"]),
			},
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"record": record, "created": created}, nil
	})

	h.registry.Register("memory.catalog.candidate.approve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleMemoryApprove)
		if err != nil {
			return nil, err
		}
		return h.catalogSvc.ApproveCandidate(stringParam(params["id"]), review)
	})

	h.registry.Register("memory.catalog.candidate.reject", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleMemoryApprove)
		if err != nil {
			return nil, err
		}
		return h.catalogSvc.RejectCandidate(stringParam(params["id"]), review)
	})

	h.registry.Register("memory.catalog.withdraw", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleMemoryApprove)
		if err != nil {
			return nil, err
		}
		return h.catalogSvc.Withdraw(stringParam(params["id"]), review)
	})

	h.registry.Register("memory.catalog.review.renew", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleMemoryApprove)
		if err != nil {
			return nil, err
		}
		return h.catalogSvc.RenewReview(
			stringParam(params["id"]),
			review,
			integerParam(params["days"], 0),
		)
	})

	h.registry.Register("memory.authority.list", func(_ string, params map[string]interface{}) (interface{}, error) {
		includeRedirected, _ := params["include_redirected"].(bool)
		return h.catalogSvc.ListAuthorities(stringParam(params["project_id"]), includeRedirected)
	})

	h.registry.Register("memory.authority.resolve", func(_ string, params map[string]interface{}) (interface{}, error) {
		return h.catalogSvc.ResolveAuthority(
			stringParam(params["project_id"]),
			stringParam(params["label"]),
		)
	})

	h.registry.Register("memory.authority.upsert", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleAuthorityManage)
		if err != nil {
			return nil, err
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.catalogSvc.UpsertAuthority(catalog.AuthorityInput{
			ID:             stringParam(params["id"]),
			ProjectID:      stringParam(params["project_id"]),
			Type:           catalog.AuthorityType(stringParam(params["type"])),
			PreferredLabel: stringParam(params["preferred_label"]),
			Aliases:        stringSliceParam(params["aliases"]),
			Description:    stringParam(params["description"]),
			ExternalIDs:    stringMapParam(params["external_ids"]),
			CreatedBy:      actor,
		}, review)
	})

	h.registry.Register("memory.authority.merge", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleAuthorityManage)
		if err != nil {
			return nil, err
		}
		return h.catalogSvc.MergeAuthority(
			stringParam(params["source_id"]),
			stringParam(params["target_id"]),
			review,
		)
	})

	h.registry.Register("memory.catalog.usage", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		kind := catalog.UsageKind(stringParam(params["kind"]))
		if err := h.catalogSvc.RecordUsage(
			stringParam(params["id"]),
			kind,
			sessionID,
			stringParam(params["trace_id"]),
			stringMapParam(params["metadata"]),
		); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "recorded"}, nil
	})
}

func integerParam(value interface{}, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func numberParam(value interface{}, fallback float64) float64 {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case float64:
		return typed
	default:
		return fallback
	}
}

func stringSlicesMapParam(value interface{}) map[string][]string {
	raw, ok := value.(map[string]interface{})
	if !ok {
		if typed, direct := value.(map[string][]string); direct {
			return typed
		}
		return nil
	}
	result := make(map[string][]string)
	for key, item := range raw {
		values := stringSliceParam(item)
		if len(values) > 0 {
			result[key] = values
		}
	}
	return result
}

func stringMapParam(value interface{}) map[string]string {
	raw, ok := value.(map[string]interface{})
	if !ok {
		if typed, direct := value.(map[string]string); direct {
			return typed
		}
		return nil
	}
	result := make(map[string]string)
	for key, item := range raw {
		text, ok := item.(string)
		if ok && strings.TrimSpace(key) != "" && strings.TrimSpace(text) != "" {
			result[strings.TrimSpace(key)] = strings.TrimSpace(text)
		}
	}
	return result
}

func timeParam(value interface{}) *time.Time {
	raw := stringParam(value)
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}
