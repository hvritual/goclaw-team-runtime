package setting

import (
	"testing"
	"time"
)

func TestSkillBindingNormalizesReferences(t *testing.T) {
	version := " version-1 "
	value, err := NewSkillBinding(
		"workspace-1", " skill-1 ", &version, true,
		map[string]any{"mode": "strict"}, []string{"agent-1", " agent-1 ", "agent-2"}, time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if value.SkillID != "skill-1" || value.SkillVersionID == nil || *value.SkillVersionID != "version-1" || len(value.AgentIDs) != 2 {
		t.Fatalf("SkillBinding = %+v", value)
	}
}
