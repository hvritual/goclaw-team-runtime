package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAcceptsLegacyStringModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "agents": {
    "defaults": {
      "model": "codex/default"
    }
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Agents.Defaults.Model.Effective(); got != "codex/default" {
		t.Fatalf("unexpected model: %s", got)
	}
}

func TestLoadAcceptsStructuredModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "agents": {
    "defaults": {
      "model": {
        "primary": "codex/default",
        "fallbacks": ["ollama/local"]
      }
    }
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Agents.Defaults.Model.Effective(); got != "codex/default" {
		t.Fatalf("unexpected model: %s", got)
	}
	if len(cfg.Agents.Defaults.Model.Fallbacks) != 1 {
		t.Fatalf("unexpected fallbacks: %+v", cfg.Agents.Defaults.Model.Fallbacks)
	}
}
