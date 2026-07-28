package ouroboros

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectRepositoryIsBoundedAndExcludesSensitiveNames(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(repo, "go.mod"),
		[]byte("module example.test/project\n\ngo 1.25.5\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".env"), []byte("SECRET=value"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "deploy-token.txt"), []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}

	resolved, context, err := inspectRepository(repo)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != repo {
		t.Fatalf("unexpected repository path %q", resolved)
	}
	if !strings.Contains(context.Manifests["go.mod"], "module example.test/project") {
		t.Fatalf("expected bounded go.mod context: %#v", context.Manifests)
	}
	for _, name := range context.TopLevel {
		if name == ".env" || name == "deploy-token.txt" {
			t.Fatalf("sensitive name leaked into model context: %s", name)
		}
	}
}

func TestReadBoundedFileDoesNotExceedLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 100)), 0o600); err != nil {
		t.Fatal(err)
	}
	content, truncated, err := readBoundedFile(path, 16)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(content) != 16 {
		t.Fatalf("expected 16-byte truncated content, got len=%d truncated=%v", len(content), truncated)
	}
}
