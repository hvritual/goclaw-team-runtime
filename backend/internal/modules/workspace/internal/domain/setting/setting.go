package setting

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid workspace setting")

type Setting struct {
	WorkspaceID, Key string
	Value            map[string]any
	UpdatedAt        time.Time
}
type SkillBinding struct {
	WorkspaceID, SkillID string
	SkillVersionID       *string
	Enabled              bool
	Configuration        map[string]any
	AgentIDs             []string
	UpdatedAt            time.Time
}

func New(workspaceID, key string, value map[string]any, now time.Time) (Setting, error) {
	workspaceID, key = strings.TrimSpace(workspaceID), strings.TrimSpace(key)
	if workspaceID == "" || key == "" {
		return Setting{}, ErrInvalid
	}
	return Setting{WorkspaceID: workspaceID, Key: key, Value: clone(value), UpdatedAt: now.UTC()}, nil
}
func NewSkillBinding(workspaceID, skillID string, version *string, enabled bool, configuration map[string]any, agentIDs []string, now time.Time) (SkillBinding, error) {
	workspaceID, skillID = strings.TrimSpace(workspaceID), strings.TrimSpace(skillID)
	if workspaceID == "" || skillID == "" {
		return SkillBinding{}, ErrInvalid
	}
	if version != nil {
		v := strings.TrimSpace(*version)
		if v == "" {
			version = nil
		} else {
			version = &v
		}
	}
	ids := make([]string, 0, len(agentIDs))
	seen := map[string]struct{}{}
	for _, id := range agentIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return SkillBinding{}, ErrInvalid
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return SkillBinding{WorkspaceID: workspaceID, SkillID: skillID, SkillVersionID: version, Enabled: enabled, Configuration: clone(configuration), AgentIDs: ids, UpdatedAt: now.UTC()}, nil
}
func clone(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for k, v := range input {
		result[k] = v
	}
	return result
}
